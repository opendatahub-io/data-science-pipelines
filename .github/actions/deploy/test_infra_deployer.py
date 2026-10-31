"""Tests for Kind CI infrastructure helpers."""

from pathlib import Path
import tempfile
import unittest

import yaml

from infra_deployer import InfraDeployer


class TestSetKustomizeNamespace(unittest.TestCase):

    def test_rewrites_opendatahub_to_rhods(self):
        with tempfile.TemporaryDirectory() as tmp:
            kustomization_path = Path(tmp) / 'kustomization.yaml'
            kustomization_path.write_text(
                'apiVersion: kustomize.config.k8s.io/v1beta1\n'
                'kind: Kustomization\n'
                'namespace: opendatahub\n'
                'resources:\n'
                '  - ../../../config/argo\n',
                encoding='utf-8',
            )

            InfraDeployer._set_kustomize_namespace(tmp, 'rhods')

            with kustomization_path.open(encoding='utf-8') as kustomization_file:
                data = yaml.safe_load(kustomization_file)
            self.assertEqual(data['namespace'], 'rhods')
            self.assertEqual(data['resources'], ['../../../config/argo'])

    def test_leaves_matching_namespace_unchanged(self):
        original = (
            'apiVersion: kustomize.config.k8s.io/v1beta1\n'
            'kind: Kustomization\n'
            'namespace: opendatahub\n'
        )
        with tempfile.TemporaryDirectory() as tmp:
            kustomization_path = Path(tmp) / 'kustomization.yaml'
            kustomization_path.write_text(original, encoding='utf-8')

            InfraDeployer._set_kustomize_namespace(tmp, 'opendatahub')

            self.assertEqual(
                kustomization_path.read_text(encoding='utf-8'), original)


if __name__ == '__main__':
    unittest.main()
