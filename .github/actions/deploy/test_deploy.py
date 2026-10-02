"""Tests for selecting and forwarding the CI database backend."""

import unittest
from types import SimpleNamespace
from unittest.mock import MagicMock
from unittest.mock import patch

from deploy import DSPDeployer


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


if __name__ == '__main__':
    unittest.main()
