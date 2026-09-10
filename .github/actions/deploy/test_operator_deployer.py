"""Tests for DSPO source selection and local image alignment."""

from pathlib import Path
import subprocess
import unittest
from unittest.mock import MagicMock, call, patch

from operator_deployer import OperatorDeployer


def _make_deployer(repo_owner='opendatahub-io',
                   target_branch='master',
                   operator_repo_owner=None,
                   operator_upstream_owner='opendatahub-io',
                   operator_branch_required=False):
    """Create an OperatorDeployer with stubbed dependencies."""
    args = MagicMock()
    args.operator_repo_owner = operator_repo_owner
    args.operator_upstream_owner = operator_upstream_owner
    args.operator_branch_required = operator_branch_required
    args.cluster_name = 'kfp-test'
    args.deploy_external_argo = False
    deployment_manager = MagicMock()
    deployment_manager.wait_for_resource.return_value = True
    deployer = OperatorDeployer(
        args=args,
        deployment_manager=deployment_manager,
        repo_owner=repo_owner,
        target_branch=target_branch,
        temp_dir='/tmp/test',
        operator_namespace='opendatahub',
    )
    return deployer


class TestOperatorSourceSelection(unittest.TestCase):

    def test_master_maps_to_operator_main(self):
        self.assertEqual(OperatorDeployer._operator_branch('master'), 'main')

    def test_stable_remains_stable(self):
        self.assertEqual(OperatorDeployer._operator_branch('stable'), 'stable')

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_clone_prefers_fork_branch(self, _):
        deployer = _make_deployer(
            target_branch='feature', operator_repo_owner='contributor')
        with patch.object(
                deployer, '_clone_from_branch', return_value=True) as clone:
            deployer.clone_operator_repo()

        clone.assert_called_once_with(
            'contributor', 'feature',
            '/tmp/test/data-science-pipelines-operator')

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_clone_falls_back_to_upstream_same_branch(self, _):
        deployer = _make_deployer(
            target_branch='feature', operator_repo_owner='contributor')
        with patch.object(
                deployer, '_clone_from_branch', side_effect=[False,
                                                             True]) as clone:
            deployer.clone_operator_repo()

        self.assertEqual(clone.call_args_list, [
            call('contributor', 'feature',
                 '/tmp/test/data-science-pipelines-operator'),
            call('opendatahub-io', 'feature',
                 '/tmp/test/data-science-pipelines-operator'),
        ])

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_clone_falls_back_to_upstream_default_branch(self, _):
        deployer = _make_deployer(
            target_branch='feature', operator_repo_owner='contributor')
        with patch.object(
                deployer, '_clone_from_branch', return_value=False):
            deployer.clone_operator_repo()

        deployer.deployment_manager.run_command.assert_called_once_with([
            'git', 'clone', '--depth', '1', '--branch', 'main',
            'https://github.com/opendatahub-io/data-science-pipelines-operator.git',
            '/tmp/test/data-science-pipelines-operator'
        ])

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_explicit_branch_does_not_fall_back(self, _):
        deployer = _make_deployer(
            target_branch='stable', operator_repo_owner='contributor',
            operator_branch_required=True)
        with patch.object(deployer, '_clone_from_branch',
                          return_value=False) as clone:
            with self.assertRaisesRegex(RuntimeError,
                                        'Required DSPO branch stable'):
                deployer.clone_operator_repo()
        clone.assert_called_once_with(
            'opendatahub-io', 'stable',
            '/tmp/test/data-science-pipelines-operator')

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_non_odh_fork_falls_back_to_canonical_upstream(self, _):
        deployer = _make_deployer(
            repo_owner='contributor', target_branch='feature',
            operator_repo_owner='contributor')
        with patch.object(
                deployer, '_clone_from_branch', return_value=False):
            deployer.clone_operator_repo()

        deployer.deployment_manager.run_command.assert_called_once_with([
            'git', 'clone', '--depth', '1', '--branch', 'main',
            'https://github.com/opendatahub-io/data-science-pipelines-operator.git',
            '/tmp/test/data-science-pipelines-operator'
        ])


class TestOperatorImageAlignment(unittest.TestCase):

    def test_build_load_and_deploy_use_same_source_image(self):
        deployer = _make_deployer(target_branch='stable')
        deployer.operator_repo_path = (
            '/tmp/test/data-science-pipelines-operator')
        deployer.deployment_manager.run_command.return_value = (
            subprocess.CompletedProcess([], 0, stdout='abc123def456\n'))

        image = deployer.build_operator_image()
        with patch.object(deployer, '_patch_params_for_kind'):
            deployer.deploy_operator()

        self.assertEqual(image, 'dspo-ci:abc123def456')
        commands = [
            command.args[0]
            for command in deployer.deployment_manager.run_command.call_args_list
        ]
        self.assertIn(['docker', 'build', '-t', image, '.'], commands)
        self.assertIn([
            'kind', 'load', 'docker-image', image, '--name', 'kfp-test'
        ], commands)
        self.assertIn(['make', 'deploy-kind', f'IMG={image}'], commands)

        deploy_call = next(
            command for command in
            deployer.deployment_manager.run_command.call_args_list
            if command.args[0][0:2] == ['make', 'deploy-kind'])
        self.assertEqual(deploy_call.kwargs['env']['IMG'], image)
        self.assertEqual(deploy_call.kwargs['env']['IMAGES_DSPO'], image)

    def test_deploy_rejects_image_not_built_from_checkout(self):
        deployer = _make_deployer()
        deployer.operator_repo_path = (
            '/tmp/test/data-science-pipelines-operator')

        with patch.object(deployer, '_patch_params_for_kind'):
            with self.assertRaisesRegex(ValueError, 'not built from cloned source'):
                deployer.deploy_operator()


class TestActionBranchWiring(unittest.TestCase):

    def _action_text(self):
        return Path(__file__).with_name('action.yml').read_text()

    def test_action_does_not_use_reserved_github_base_ref(self):
        action = self._action_text()

        self.assertNotIn('GITHUB_BASE_REF', action)
        self.assertIn('OPERATOR_BRANCH:', action)
        self.assertIn('--operator-branch "$OPERATOR_BRANCH"', action)

    def test_operator_deployment_remains_opt_in_by_default(self):
        action = self._action_text()
        operator_input = action.split('skip_operator_deployment:', 1)[1]
        operator_input = operator_input.split('deploy_external_db:', 1)[0]

        self.assertIn("default: 'true'", operator_input)


if __name__ == '__main__':
    unittest.main()
