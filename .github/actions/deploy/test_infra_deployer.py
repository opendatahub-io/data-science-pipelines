"""Tests for kustomize namespace handling in InfraDeployer."""

import os
import shutil
import tempfile
import unittest
from unittest.mock import MagicMock

from infra_deployer import InfraDeployer
import yaml


def _make_deployer(operator_namespace='rhods',
                   pipeline_store='kubernetes',
                   operator_repo_path=None):
    """Create an InfraDeployer with stubbed dependencies."""
    args = MagicMock()
    args.pipeline_store = pipeline_store
    root = operator_repo_path or tempfile.mkdtemp(prefix='dspo-test-')
    deployer = InfraDeployer(
        args=args,
        deployment_manager=MagicMock(),
        operator_repo_path=os.path.join(root,
                                        'data-science-pipelines-operator'),
        deployment_namespace='kubeflow',
        dspa_name='dspa-test',
        operator_namespace=operator_namespace,
        temp_dir=tempfile.mkdtemp(prefix='infra-test-'))
    os.makedirs(deployer.operator_repo_path, exist_ok=True)
    return deployer


def _make_kustomization(root, name, namespace='opendatahub'):
    """Create a kustomization directory inside a fake DSPO checkout."""
    path = os.path.join(root, name)
    os.makedirs(path)
    with open(os.path.join(path, 'kustomization.yaml'), 'w') as handle:
        yaml.safe_dump({'namespace': namespace}, handle)
    return path


class TestApplyKustomizeWithNamespace(unittest.TestCase):

    def setUp(self):
        self.deployer = _make_deployer()
        self.manifest_path = _make_kustomization(
            self.deployer.operator_repo_path, 'argo-lite')

    def tearDown(self):
        shutil.rmtree(self.deployer.temp_dir, ignore_errors=True)
        shutil.rmtree(
            os.path.dirname(self.deployer.operator_repo_path),
            ignore_errors=True)

    def test_wrapper_sets_namespace_and_relative_resource(self):
        self.deployer.apply_kustomize_with_namespace(self.manifest_path,
                                                     'rhods', 'Argo Lite')

        wrapper_dir = os.path.join(
            os.path.dirname(self.manifest_path), 'argo-lite-ns-rhods')
        # Wrapper must be removed after the apply.
        self.assertFalse(os.path.exists(wrapper_dir))

    def test_apply_resource_called_with_wrapper_and_no_namespace(self):
        self.deployer.apply_kustomize_with_namespace(self.manifest_path,
                                                     'rhods', 'Argo Lite')

        apply_call = self.deployer.deployment_manager.apply_resource.call_args
        wrapper_dir = apply_call.kwargs['manifest_path']
        self.assertTrue(apply_call.kwargs['kustomize'])
        self.assertNotIn('namespace', apply_call.kwargs)
        self.assertEqual(apply_call.kwargs['description'], 'Argo Lite')
        # The wrapper is a sibling of the kustomization it wraps.
        self.assertEqual(
            os.path.dirname(wrapper_dir), os.path.dirname(self.manifest_path))

    def test_wrapper_content_sets_target_namespace(self):
        captured = {}

        def capture(manifest_path=None, **kwargs):
            captured['kustomization'] = yaml.safe_load(
                open(os.path.join(manifest_path, 'kustomization.yaml')))

        self.deployer.deployment_manager.apply_resource.side_effect = capture
        self.deployer.apply_kustomize_with_namespace(self.manifest_path,
                                                     'rhods', 'Argo Lite')

        self.assertEqual(captured['kustomization']['namespace'], 'rhods')
        self.assertEqual(captured['kustomization']['resources'],
                         [os.path.join('..', 'argo-lite')])

    def test_wrapper_cleaned_up_when_apply_fails(self):
        self.deployer.deployment_manager.apply_resource.side_effect = (
            RuntimeError('apply failed'))
        wrapper_dir = os.path.join(
            os.path.dirname(self.manifest_path), 'argo-lite-ns-rhods')

        with self.assertRaises(RuntimeError):
            self.deployer.apply_kustomize_with_namespace(
                self.manifest_path, 'rhods', 'Argo Lite')

        self.assertFalse(os.path.exists(wrapper_dir))

    def test_repeated_apply_reuses_distinct_wrapper_names(self):
        # Applying for two different namespaces must not collide on the
        # wrapper directory name.
        seen = []

        def capture(manifest_path=None, **kwargs):
            seen.append(manifest_path)

        self.deployer.deployment_manager.apply_resource.side_effect = capture
        self.deployer.apply_kustomize_with_namespace(self.manifest_path,
                                                     'rhods', 'Argo Lite')
        self.deployer.apply_kustomize_with_namespace(self.manifest_path,
                                                     'opendatahub', 'Argo Lite')

        self.assertEqual(len(set(seen)), 2)


class TestDeployArgoLite(unittest.TestCase):

    def setUp(self):
        self.deployer = _make_deployer()
        self.argo_lite = _make_kustomization(self.deployer.operator_repo_path,
                                             'argo-lite')

    def tearDown(self):
        shutil.rmtree(
            os.path.dirname(self.deployer.operator_repo_path),
            ignore_errors=True)

    def test_uses_namespace_wrapper_instead_of_dash_n(self):
        with patch_os_path_exists(True):
            self.deployer.deploy_argo_lite()

        apply_call = self.deployer.deployment_manager.apply_resource.call_args
        self.assertTrue(apply_call.kwargs['kustomize'])
        self.assertNotIn('namespace', apply_call.kwargs)


class TestApplyWebhooks(unittest.TestCase):

    def setUp(self):
        self.deployer = _make_deployer()
        self.webhook = _make_kustomization(self.deployer.operator_repo_path,
                                           'webhook')

    def tearDown(self):
        shutil.rmtree(
            os.path.dirname(self.deployer.operator_repo_path),
            ignore_errors=True)

    def test_uses_namespace_wrapper_instead_of_dash_n(self):
        with patch_os_path_exists(True):
            self.deployer.apply_webhooks()

        apply_call = self.deployer.deployment_manager.apply_resource.call_args
        self.assertTrue(apply_call.kwargs['kustomize'])
        self.assertNotIn('namespace', apply_call.kwargs)

    def test_skipped_for_non_kubernetes_store(self):
        deployer = _make_deployer(pipeline_store='database')
        deployer.apply_webhooks()
        deployer.deployment_manager.apply_resource.assert_not_called()


def patch_os_path_exists(value):
    """Patch infra_deployer.os.path.exists for the DSPO checkout layout."""
    from unittest.mock import patch

    real_exists = os.path.exists

    def exists(path):
        if path.endswith(('argo-lite', 'webhook')):
            return value
        return real_exists(path)

    return patch('infra_deployer.os.path.exists', side_effect=exists)


if __name__ == '__main__':
    unittest.main()
