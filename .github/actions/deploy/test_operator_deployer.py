"""Tests for OperatorDeployer operator-image tag and registry resolution."""

import unittest
from unittest.mock import MagicMock, patch

from operator_deployer import OperatorDeployer


def _make_deployer(repo_owner='opendatahub-io',
                   target_branch='master',
                   operator_image_tag=''):
    """Create an OperatorDeployer with stubbed dependencies."""
    args = MagicMock()
    args.operator_image_tag = operator_image_tag
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
    deployer.operator_repo_path = '/tmp/test/data-science-pipelines-operator'
    return deployer


def _resolve_operator_image(deployer):
    """Call deploy_operator and return the IMG value passed to make."""
    with patch.object(deployer, '_patch_params_for_kind'):
        deployer.deploy_operator()
    for call in deployer.deployment_manager.run_command.call_args_list:
        args = call[0][0]
        for arg in args:
            if isinstance(arg, str) and arg.startswith('IMG='):
                return arg.split('=', 1)[1]
    raise AssertionError('No IMG= argument found in run_command calls')


class TestOperatorImageResolution(unittest.TestCase):

    def test_odh_master_uses_odh_main_tag(self):
        deployer = _make_deployer(
            repo_owner='opendatahub-io', target_branch='master')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:odh-main')

    def test_odh_stable_uses_odh_stable_tag(self):
        deployer = _make_deployer(
            repo_owner='opendatahub-io', target_branch='stable')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:odh-stable')

    def test_rhds_master_uses_main_tag(self):
        deployer = _make_deployer(
            repo_owner='red-hat-data-services', target_branch='master')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:main')

    def test_rhds_rhoai_branch_uses_branch_name_as_tag(self):
        deployer = _make_deployer(
            repo_owner='red-hat-data-services', target_branch='rhoai-2.16')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:rhoai-2.16')

    def test_explicit_tag_overrides_branch_logic(self):
        deployer = _make_deployer(
            repo_owner='opendatahub-io',
            target_branch='master',
            operator_image_tag='custom-tag')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:custom-tag')

    def test_odh_unknown_branch_uses_branch_name_as_tag(self):
        deployer = _make_deployer(
            repo_owner='opendatahub-io', target_branch='feature-x')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:feature-x')

    def test_rhds_stable_uses_odh_stable_tag(self):
        deployer = _make_deployer(
            repo_owner='red-hat-data-services', target_branch='stable')
        image = _resolve_operator_image(deployer)
        self.assertEqual(
            image,
            'quay.io/opendatahub/data-science-pipelines-operator:odh-stable')


if __name__ == '__main__':
    unittest.main()
