"""Certificate coverage for Kind operator deployments."""

from types import SimpleNamespace
import unittest
from unittest.mock import MagicMock
from unittest.mock import patch

from tls_manager import TLSCertificateManager
import yaml


def _make_manager(pod_tls=False, external_db=False):
    deployer = SimpleNamespace(
        args=SimpleNamespace(
            pod_to_pod_tls_enabled=pod_tls, deploy_external_db=external_db),
        dspa_name='test-dspa',
        deployment_namespace='test-ns',
        operator_namespace='opendatahub',
        external_db_namespace='external-db',
        deployment_manager=MagicMock(),
    )
    return TLSCertificateManager(deployer)


class TestKindCertificates(unittest.TestCase):

    def test_diagnostics_limit_output_and_do_not_print_secret_data(self):
        manager = _make_manager(pod_tls=True, external_db=True)
        manager.debug_certificates()
        calls = manager.deployer.deployment_manager.run_command.call_args_list
        commands = [call.args[0] for call in calls]
        secret_commands = [
            command for command in commands
            if command[1:3] == ['get', 'secrets']
        ]
        self.assertEqual(len(secret_commands), 2)
        for command in secret_commands:
            self.assertIn('custom-columns=NAME:.metadata.name,TYPE:.type',
                          command)
        for command_call in calls:
            self.assertFalse(command_call.kwargs['check'])
            self.assertEqual(command_call.kwargs['timeout'], 20)
            self.assertIn('--request-timeout=10s', command_call.args[0])
        logs = [command for command in commands if command[1] == 'logs']
        self.assertEqual(len(logs), 2)
        for command in logs:
            self.assertIn('--tail=100', command)

    def test_diagnostics_continue_after_a_command_timeout(self):
        manager = _make_manager()
        runner = manager.deployer.deployment_manager.run_command
        runner.side_effect = [TimeoutError('timeout')] + [MagicMock()] * 8
        manager.debug_certificates()
        self.assertEqual(runner.call_count, 9)

    def test_proxy_certificates_are_required_in_every_tls_mode(self):
        for pod_tls in (False, True):
            for external_db in (False, True):
                with self.subTest(pod_tls=pod_tls, external_db=external_db):
                    manager = _make_manager(pod_tls, external_db)
                    certificates = manager._get_required_certificates()
                    by_name = {cert['name']: cert for cert in certificates}
                    proxy_names = {
                        'ds-pipelines-proxy-tls-test-dspa',
                        'ds-pipelines-envoy-proxy-tls-test-dspa',
                    }
                    self.assertTrue(proxy_names <= by_name.keys())
                    for name in proxy_names:
                        self.assertEqual(by_name[name]['namespace'], 'test-ns')
                    if pod_tls:
                        self.assertEqual(len(certificates), 4)
                        database = by_name['ds-pipelines-mariadb-tls-test-dspa']
                        self.assertEqual(
                            database['namespace'],
                            'external-db' if external_db else 'test-ns')
                    else:
                        self.assertEqual(by_name.keys(), proxy_names)

    def test_setup_creates_proxy_certificates_without_enabling_database_tls(
            self):
        manager = _make_manager()
        with patch.object(manager, '_create_service_ca_issuer'), \
             patch.object(manager, '_create_service_ca_certificate'), \
             patch.object(manager, '_create_ca_issuer_from_service_ca'), \
             patch.object(manager, '_create_service_ca_configmap'), \
             patch.object(manager, '_verify_certificates'), \
             patch.object(manager, '_create_component_certificate') as create, \
             patch.object(manager, '_configure_internal_mariadb_tls') as database:
            manager.setup_tls_for_kind()

        self.assertEqual({call.args[0] for call in create.call_args_list}, {
            'ds-pipelines-proxy-tls-test-dspa',
            'ds-pipelines-envoy-proxy-tls-test-dspa',
        })
        database.assert_not_called()

    def test_ca_issuer_uses_only_namespaces_requiring_certificates(self):
        for pod_tls in (False, True):
            with self.subTest(pod_tls=pod_tls):
                manager = _make_manager(pod_tls, external_db=True)
                with patch.object(manager, '_copy_secret_between_namespaces'), \
                     patch.object(manager, '_apply_yaml_content') as apply:
                    manager._create_ca_issuer_from_service_ca()
                namespaces = {
                    yaml.safe_load(call.args[0])['metadata']['namespace']
                    for call in apply.call_args_list
                }
                self.assertEqual(
                    namespaces,
                    {'test-ns', 'external-db'} if pod_tls else {'test-ns'})


if __name__ == '__main__':
    unittest.main()
