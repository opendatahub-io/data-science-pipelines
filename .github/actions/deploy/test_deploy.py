"""Tests for DSP deployment helper behavior."""

from types import SimpleNamespace
import unittest
from unittest.mock import call
from unittest.mock import MagicMock
from unittest.mock import patch

from deploy import DSPDeployer
from tls_manager import TLSCertificateManager


class TestDatabaseDeployment(unittest.TestCase):

    def setUp(self):
        self.args = SimpleNamespace(
            db_type='mysql',
            proxy=False,
            multi_user=False,
            cache_enabled=True,
            pipeline_store='database',
            artifact_proxy=False,
            storage_backend='seaweedfs',
            argo_version=None,
            pod_to_pod_tls_enabled=False,
            image_registry='kind-registry:5000',
            image_tag='ci',
            enable_modular_architecture=False,
        )
        self.deployer = DSPDeployer(self.args)
        self.deployer.deployment_manager = MagicMock()

    def test_mysql_retains_operator_deployment(self):
        self.assertTrue(self.deployer._should_use_operator_deployment())

    def test_postgres_uses_direct_deployment(self):
        self.args.db_type = 'pgx'
        self.assertFalse(self.deployer._should_use_operator_deployment())

    def test_direct_deployment_forwards_database_driver(self):
        for database_driver in ('mysql', 'pgx'):
            with self.subTest(database_driver=database_driver):
                self.args.db_type = database_driver
                self.deployer.deploy_dsp_direct()
                command = self.deployer.deployment_manager.run_command.call_args.args[0]
                self.assertEqual(command[-2:], ['--db-type', database_driver])

    @patch('deploy.output_to_github_actions')
    def test_postgres_metadata_uses_postgres_service(self, output):
        self.args.db_type = 'pgx'
        self.deployer.dspa = MagicMock()
        self.deployer.dspa.output_deployment_metadata.return_value = {
            'DATABASE_NAME': 'mysql',
        }
        self.deployer.output_deployment_metadata()
        output.assert_called_once_with('DATABASE_NAME', 'postgres')


class TestOperatorCertificateSetup(unittest.TestCase):

    def test_failed_deployment_collects_diagnostics_and_preserves_error(self):
        deployer = DSPDeployer(
            SimpleNamespace(
                skip_operator_deployment=False,
                multi_user=False,
                proxy=False,
                enable_modular_architecture=False))
        deployer.operator = MagicMock()
        deployer.operator.clone_operator_repo.side_effect = RuntimeError(
            'original deployment failure')
        deployer.tls_manager = MagicMock()
        deployer.tls_manager.debug_certificates.side_effect = RuntimeError(
            'diagnostic failure')
        with patch.object(deployer, 'setup_environment'):
            with self.assertRaisesRegex(RuntimeError,
                                        'original deployment failure'):
                deployer.deploy()
        deployer.tls_manager.debug_certificates.assert_called_once_with()

    def test_tls_disabled_operator_deployment_creates_proxy_certificates(self):
        args = SimpleNamespace(
            skip_operator_deployment=False,
            multi_user=False,
            proxy=False,
            enable_modular_architecture=False,
            pod_to_pod_tls_enabled=False,
            pipeline_store='database',
            deploy_external_db=False,
            github_repository='opendatahub-io/data-science-pipelines',
            operator_branch='main',
            namespace='test-ns',
            dspa_name='test-dspa',
        )
        deployer = DSPDeployer(args)
        deployer.deployment_manager = MagicMock()
        deployer.deployment_manager.wait_for_resource.return_value = True
        with patch(
                'deploy.tempfile.mkdtemp', return_value='/tmp/pr465-unit-test'):
            deployer.setup_environment()
        self.assertIsInstance(deployer.tls_manager, TLSCertificateManager)
        deploy_cert_manager = deployer.infra.deploy_cert_manager

        deployer.temp_dir = None
        deployer.operator = MagicMock()
        deployer.infra = MagicMock()
        deployer.infra.deploy_cert_manager.side_effect = deploy_cert_manager
        deployer.database = MagicMock()
        deployer.storage = MagicMock()
        deployer.dspa = MagicMock()
        with patch.object(deployer, 'setup_environment'), \
             patch.object(deployer, '_init_deployers_after_clone'), \
             patch.object(deployer, '_patch_v1_allowed_namespaces'), \
             patch.object(deployer.tls_manager, 'setup_tls_for_kind') as setup:
            setup_order = MagicMock()
            setup_order.attach_mock(deployer.deployment_manager.apply_resource,
                                    'install')
            setup_order.attach_mock(
                deployer.deployment_manager.wait_for_resource, 'wait')
            setup_order.attach_mock(setup, 'certificates')
            deployer.deploy()
        setup.assert_called_once_with()
        deployer.deployment_manager.apply_resource.assert_called_once_with(
            manifest_path='https://github.com/cert-manager/cert-manager/releases/download/v1.16.3/cert-manager.yaml',
            description='cert-manager')
        self.assertEqual([entry[0] for entry in setup_order.mock_calls],
                         ['install', 'wait', 'wait', 'certificates'])
        deployer.dspa.deploy_dsp_via_operator.assert_called_once_with()


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
