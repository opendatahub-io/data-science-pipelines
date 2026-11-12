"""Tests for SeaweedFS network policy updates in StorageDeployer."""

import shutil
import tempfile
import unittest
from unittest.mock import MagicMock

from storage_deployer import StorageDeployer
import yaml


def _make_deployer(storage_backend='seaweedfs',
                   deployment_namespace='kubeflow',
                   operator_namespace='rhods'):
    """Create a StorageDeployer with stubbed dependencies."""
    args = MagicMock()
    args.storage_backend = storage_backend
    temp_dir = tempfile.mkdtemp(prefix='storage-test-')
    return StorageDeployer(
        args=args,
        deployment_manager=MagicMock(),
        deployment_namespace=deployment_namespace,
        temp_dir=temp_dir,
        operator_namespace=operator_namespace), temp_dir


class TestUpdateSeaweedfsNetworkPolicy(unittest.TestCase):

    def setUp(self):
        self.deployer, self.temp_dir = _make_deployer()
        self.captured = {}

        def capture(manifest_path=None, **kwargs):
            with open(manifest_path) as handle:
                self.captured['policy'] = yaml.safe_load(handle)

        self.deployer.deployment_manager.apply_resource.side_effect = capture

    def tearDown(self):
        shutil.rmtree(self.temp_dir, ignore_errors=True)

    def test_allows_operator_namespace(self):
        self.deployer.update_seaweedfs_network_policy()

        allowed = self.captured['policy']['spec']['ingress'][1]['from'][0][
            'namespaceSelector']['matchExpressions'][0]['values']
        self.assertIn('rhods', allowed)

    def test_allows_opendatahub_when_operator_runs_there(self):
        deployer, temp_dir = _make_deployer(operator_namespace='opendatahub')
        try:
            captured = {}

            def capture(manifest_path=None, **kwargs):
                with open(manifest_path) as handle:
                    captured['policy'] = yaml.safe_load(handle)

            deployer.deployment_manager.apply_resource.side_effect = capture
            deployer.update_seaweedfs_network_policy()

            allowed = captured['policy']['spec']['ingress'][1]['from'][0][
                'namespaceSelector']['matchExpressions'][0]['values']
            self.assertIn('opendatahub', allowed)
        finally:
            shutil.rmtree(temp_dir, ignore_errors=True)

    def test_applied_into_deployment_namespace(self):
        self.deployer.update_seaweedfs_network_policy()

        apply_call = self.deployer.deployment_manager.apply_resource.call_args
        self.assertEqual(apply_call.kwargs['namespace'], 'kubeflow')


class TestApplySeaweedfsNetworkingFixes(unittest.TestCase):

    def test_skipped_for_non_seaweedfs_backend(self):
        deployer, temp_dir = _make_deployer(storage_backend='minio')
        try:
            deployer.apply_seaweedfs_networking_fixes()
            deployer.deployment_manager.apply_resource.assert_not_called()
        finally:
            shutil.rmtree(temp_dir, ignore_errors=True)


if __name__ == '__main__':
    unittest.main()
