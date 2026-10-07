"""Tests for DSPO source selection and local image alignment."""

from pathlib import Path
import subprocess
import unittest
from unittest.mock import call
from unittest.mock import MagicMock
from unittest.mock import patch

from operator_deployer import OperatorDeployer
import yaml


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
        with patch.object(deployer, '_clone_from_branch', return_value=False):
            deployer.clone_operator_repo()

        deployer.deployment_manager.run_command.assert_called_once_with([
            'git', 'clone', '--depth', '1', '--branch', 'main',
            'https://github.com/opendatahub-io/data-science-pipelines-operator.git',
            '/tmp/test/data-science-pipelines-operator'
        ])

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_explicit_branch_does_not_fall_back(self, _):
        deployer = _make_deployer(
            target_branch='stable',
            operator_repo_owner='contributor',
            operator_branch_required=True)
        with patch.object(
                deployer, '_clone_from_branch', return_value=False) as clone:
            with self.assertRaisesRegex(RuntimeError,
                                        'Required DSPO branch stable'):
                deployer.clone_operator_repo()
        clone.assert_called_once_with(
            'opendatahub-io', 'stable',
            '/tmp/test/data-science-pipelines-operator')

    @patch('operator_deployer.os.path.exists', return_value=False)
    def test_non_odh_fork_falls_back_to_canonical_upstream(self, _):
        deployer = _make_deployer(
            repo_owner='contributor',
            target_branch='feature',
            operator_repo_owner='contributor')
        with patch.object(deployer, '_clone_from_branch', return_value=False):
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
            command.args[0] for command in
            deployer.deployment_manager.run_command.call_args_list
        ]
        self.assertIn(['docker', 'build', '-t', image, '.'], commands)
        self.assertIn(
            ['kind', 'load', 'docker-image', image, '--name', 'kfp-test'],
            commands)
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
            with self.assertRaisesRegex(ValueError,
                                        'not built from cloned source'):
                deployer.deploy_operator()

    def test_enable_modular_architecture_applies_fixture_and_waits_for_readiness(
            self):
        deployer = _make_deployer()
        deployer.operator_repo_path = '/tmp/test/data-science-pipelines-operator'
        deployer.deployment_manager.run_command.return_value = (
            subprocess.CompletedProcess([],
                                        0,
                                        stdout='''apiVersion: v1
kind: ConfigMap
metadata:
  name: odh-aipipelines-config
  namespace: opendatahub
data:
  platformVersion: kind-modular-ci
---
apiVersion: components.platform.opendatahub.io/v1alpha1
kind: AIPipelines
metadata:
  name: default-aipipelines
'''))

        platform_version = deployer.enable_modular_architecture()

        commands = [
            command.args[0] for command in
            deployer.deployment_manager.run_command.call_args_list
        ]
        self.assertEqual(commands, [
            [
                'kubectl', 'kustomize',
                '/tmp/test/data-science-pipelines-operator/.github/resources/aipipelines'
            ],
            [
                'kubectl',
                'set',
                'env',
                '-n',
                'opendatahub',
                'deployment/data-science-pipelines-operator-controller-manager',
                'DSPO_ENABLEAIPIPELINESMODULECONTROLLER=true',
                'APPLICATIONS_NAMESPACE=opendatahub',
            ],
            [
                'kubectl',
                'rollout',
                'status',
                '-n',
                'opendatahub',
                'deployment/data-science-pipelines-operator-controller-manager',
                '--timeout=300s',
            ],
            [
                'kubectl',
                'wait',
                'aipipelines/default-aipipelines',
                '--for=condition=Ready=true',
                '--timeout=300s',
            ],
            [
                'kubectl',
                'wait',
                'aipipelines/default-aipipelines',
                '--for=condition=ProvisioningSucceeded=true',
                '--timeout=300s',
            ],
        ])
        applied_fixture = yaml.safe_load_all(
            deployer.deployment_manager.apply_resource.call_args
            .kwargs['manifest_content'])
        fixture_resources = list(applied_fixture)
        self.assertEqual(fixture_resources[0]['metadata']['namespace'],
                         'opendatahub')
        self.assertEqual(platform_version, 'kind-modular-ci')

    def test_enable_modular_architecture_rejects_missing_platform_version(self):
        deployer = _make_deployer()
        deployer.operator_repo_path = '/tmp/test/data-science-pipelines-operator'
        deployer.deployment_manager.run_command.return_value = (
            subprocess.CompletedProcess([],
                                        0,
                                        stdout='''apiVersion: v1
kind: ConfigMap
metadata:
  name: odh-aipipelines-config
  namespace: opendatahub
'''))

        with self.assertRaisesRegex(ValueError, 'no non-empty platformVersion'):
            deployer.enable_modular_architecture()

    def test_enable_modular_architecture_rejects_invalid_platform_config(self):
        invalid_fixtures = {
            'null data':
                '''apiVersion: v1
kind: ConfigMap
metadata:
  name: odh-aipipelines-config
  namespace: opendatahub
data:
''',
            'blank platform version':
                '''apiVersion: v1
kind: ConfigMap
metadata:
  name: odh-aipipelines-config
  namespace: opendatahub
data:
  platformVersion: ""
''',
        }

        for fixture_name, rendered_fixture in invalid_fixtures.items():
            with self.subTest(fixture_name):
                deployer = _make_deployer()
                deployer.operator_repo_path = (
                    '/tmp/test/data-science-pipelines-operator')
                deployer.deployment_manager.run_command.return_value = (
                    subprocess.CompletedProcess([], 0, stdout=rendered_fixture))

                with self.assertRaisesRegex(ValueError,
                                            'no non-empty platformVersion'):
                    deployer.enable_modular_architecture()
                deployer.deployment_manager.apply_resource.assert_not_called()

    def test_enable_modular_architecture_uses_rhods_for_platform_config(self):
        deployer = _make_deployer()
        deployer.operator_namespace = 'rhods'
        deployer.operator_repo_path = '/tmp/test/data-science-pipelines-operator'
        deployer.deployment_manager.run_command.return_value = (
            subprocess.CompletedProcess([],
                                        0,
                                        stdout='''apiVersion: v1
kind: ConfigMap
metadata:
  name: odh-aipipelines-config
  namespace: opendatahub
data:
  platformVersion: kind-modular-ci
'''))

        deployer.enable_modular_architecture()

        fixture_resources = list(
            yaml.safe_load_all(deployer.deployment_manager.apply_resource
                               .call_args.kwargs['manifest_content']))
        self.assertEqual(fixture_resources[0]['metadata']['namespace'], 'rhods')


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

    def test_aipipelines_module_is_opt_in_and_forwarded_to_deployer(self):
        action = self._action_text()
        module_input = action.split('enable_modular_architecture:', 1)[1]
        module_input = module_input.split('skip_load_docker_images:', 1)[0]

        self.assertIn("default: 'false'", module_input)
        self.assertIn(
            'ENABLE_MODULAR_ARCHITECTURE: ${{ inputs.enable_modular_architecture }}',
            action)
        self.assertIn(
            '--enable-modular-architecture "$ENABLE_MODULAR_ARCHITECTURE"',
            action)

    def test_action_exports_modular_platform_release_version(self):
        action = self._action_text()

        self.assertIn('expected_platform_release_version:', action)
        self.assertIn(
            '${{ steps.deploy-kfp.outputs.EXPECTED_PLATFORM_RELEASE_VERSION }}',
            action)

    def test_test_and_report_action_maps_expected_platform_release_version(
            self):
        action = (Path(__file__).parents[1] / 'test-and-report' /
                  'action.yml').read_text()

        self.assertIn(
            'EXPECTED_PLATFORM_RELEASE_VERSION: '
            '${{ inputs.expected_platform_release_version }}', action)

    def test_upgrade_workflow_enables_module_only_for_target_deployment(self):
        workflow = (Path(__file__).parents[2] / 'workflows' /
                    'upgrade-test.yml').read_text()
        initial_deployment, target_deployment = workflow.split(
            '      - name: Deploy from Branch', 1)

        self.assertNotIn('enable_modular_architecture:', initial_deployment)
        self.assertIn("enable_modular_architecture: 'true'", target_deployment)

    def test_upgrade_workflow_verifies_fixture_platform_release_version(self):
        workflow = (Path(__file__).parents[2] / 'workflows' /
                    'upgrade-test.yml').read_text()

        self.assertIn(
            'expected_platform_release_version: '
            '${{ steps.deploy.outputs.expected_platform_release_version }}',
            workflow)

    def test_upgrade_workflow_asserts_legacy_architecture_before_preparation(
            self):
        workflow = (Path(__file__).parents[2] / 'workflows' /
                    'upgrade-test.yml').read_text()
        legacy_checks, _ = workflow.split('      - name: Prepare for Upgrade',
                                          1)

        self.assertIn('      - name: Verify initial release is non-modular',
                      legacy_checks)
        self.assertIn(
            'kubectl get crd aipipelines.components.platform.opendatahub.io',
            legacy_checks)
        self.assertIn(
            'kubectl get aipipelines/default-aipipelines --ignore-not-found -o name',
            legacy_checks)


if __name__ == '__main__':
    unittest.main()
