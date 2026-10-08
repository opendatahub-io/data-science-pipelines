"""Tests for DSP deployment metadata output."""

from types import SimpleNamespace
import unittest
from unittest.mock import call
from unittest.mock import MagicMock
from unittest.mock import patch

from deploy import DSPDeployer


class TestExpectedPlatformReleaseVersionOutput(unittest.TestCase):

    def _make_deployer(self):
        args = SimpleNamespace(
            skip_operator_deployment=False,
            multi_user=False,
            proxy=False,
            enable_modular_architecture=False,
        )
        deployer = DSPDeployer(args)
        deployer.dspa = MagicMock()
        deployer.dspa.output_deployment_metadata.return_value = {
            'DEPLOYMENT_NAME': 'ds-pipeline-dspa-test',
        }
        return deployer

    def test_outputs_expected_platform_release_version(self):
        deployer = self._make_deployer()
        deployer.expected_platform_release_version = 'kind-modular-ci'

        with patch('deploy.output_to_github_actions') as output_metadata:
            deployer.output_deployment_metadata()

        self.assertIn(
            call('EXPECTED_PLATFORM_RELEASE_VERSION', 'kind-modular-ci'),
            output_metadata.call_args_list)

    def test_omits_expected_platform_release_version_when_unset(self):
        deployer = self._make_deployer()

        with patch('deploy.output_to_github_actions') as output_metadata:
            deployer.output_deployment_metadata()

        output_keys = [call.args[0] for call in output_metadata.call_args_list]
        self.assertNotIn('EXPECTED_PLATFORM_RELEASE_VERSION', output_keys)


class TestModularArchitectureModeSelection(unittest.TestCase):

    def test_rejects_modular_architecture_for_direct_deployment_modes(self):
        direct_modes = [
            ('skip operator deployment', {
                'skip_operator_deployment': True
            }),
            ('multi-user deployment', {
                'multi_user': True
            }),
            ('proxy deployment', {
                'proxy': True
            }),
        ]

        for mode_name, mode_arguments in direct_modes:
            with self.subTest(mode_name):
                arguments = {
                    'skip_operator_deployment': False,
                    'multi_user': False,
                    'proxy': False,
                    'enable_modular_architecture': True,
                }
                arguments.update(mode_arguments)
                args = SimpleNamespace(**arguments)
                deployer = DSPDeployer(args)
                deployer.skip_operator_deployment = (
                    args.skip_operator_deployment)

                with self.assertRaisesRegex(
                        ValueError,
                        'Modular AIPipelines requires operator deployment'):
                    deployer._should_use_operator_deployment()


if __name__ == '__main__':
    unittest.main()
