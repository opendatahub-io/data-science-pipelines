"""Infrastructure deployment (Argo, PyPI, cert-manager, RBAC, port forwarding).

Groups infrastructure concerns that support the main DSP deployment
but are not specific to a single domain like storage or database.
"""

import os
import subprocess

import yaml

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType
from deployment_manager import WaitCondition


class InfraDeployer:

    def __init__(self, args, deployment_manager: K8sDeploymentManager,
                 operator_repo_path: str, deployment_namespace: str,
                 dspa_name: str, operator_namespace: str, temp_dir: str):
        self.args = args
        self.deployment_manager = deployment_manager
        self.operator_repo_path = operator_repo_path
        self.deployment_namespace = deployment_namespace
        self.dspa_name = dspa_name
        self.operator_namespace = operator_namespace
        self.temp_dir = temp_dir

    def deploy_cert_manager(self):
        """Deploy cert-manager for certificate management."""
        if self.args.pipeline_store != 'kubernetes' and not self.args.pod_to_pod_tls_enabled:
            print(
                '🏷️ Skipping cert-manager deployment because pipeline_store != kubernetes and pod_to_pod_tls is disabled'
            )
            return

        cert_manager_namespace = 'cert-manager'

        print(f'🏷️  Creating cert-manager namespace: {cert_manager_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', cert_manager_namespace],
            check=False)

        self.deployment_manager.apply_resource(
            manifest_path='https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml',
            description='cert-manager')

        success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.POD,
            namespace=cert_manager_namespace,
            condition=WaitCondition.READY,
            timeout='120s',
            all_resources=True,
            description='cert-manager pods')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=cert_manager_namespace,
                deployment_name='cert-manager',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('Cert-manager deployment failed')

        print('✅ Cert-manager deployed successfully')

    def apply_webhooks(self):
        """Apply webhook certificates for TLS communication."""
        if self.args.pipeline_store != 'kubernetes':
            return

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for webhook certificates')

        print('📜 Applying webhook certificates for TLS communication...')

        webhook_certs_path = os.path.join(self.operator_repo_path, '.github',
                                          'resources', 'webhook')

        if os.path.exists(webhook_certs_path):
            self.deployment_manager.apply_resource(
                manifest_path=webhook_certs_path,
                namespace=self.operator_namespace,
                kustomize=True,
                description='Webhook certificates for TLS communication')
            print('✅ Webhook certificates applied for TLS communication')
        else:
            print(
                f'⚠️  Webhook certificates path not found: {webhook_certs_path}'
            )

    def deploy_argo_lite(self):
        """Deploy Argo Lite using operator repository resources."""
        print('🔧 Deploying Argo Lite...')

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for Argo Lite deployment')

        argo_lite_path = os.path.join(self.operator_repo_path, '.github',
                                      'resources', 'argo-lite')

        if not os.path.exists(argo_lite_path):
            raise ValueError(
                f'Argo Lite resources directory not found: {argo_lite_path}')

        self.deployment_manager.apply_resource(
            manifest_path=argo_lite_path,
            namespace=self.operator_namespace,
            kustomize=True,
            description='Argo Lite')

        print('✅ Argo Lite deployed successfully')

    def deploy_external_argo(self):
        """Deploy Argo Workflows externally using local manifests."""
        if not self.args.deploy_external_argo:
            return

        print(
            '⚙️  Deploying Argo Workflows externally using local manifests...')

        argo_version = self.args.argo_version or 'v3.6.7'

        print(
            f'📝 NOTE: Argo version {argo_version} specified, updating Argo Workflow manifests...'
        )

        version_file = './manifests/kustomize/third-party/argo/VERSION'
        with open(version_file, 'w') as f:
            f.write(argo_version + '\n')
        print(f'📄 Written {argo_version} to {version_file}')

        print('🔄 Updating Argo manifests...')
        self.deployment_manager.run_command([
            'make', '-C', './manifests/kustomize/third-party/argo', 'update'
        ])
        print(f'✅ Manifests updated for Argo version {argo_version}')

        print('📦 Applying Argo CRDs from local manifests...')
        crds_path = './manifests/kustomize/third-party/argo/installs/namespace/cluster-scoped'

        self.deployment_manager.apply_resource(
            manifest_path=crds_path,
            kustomize=True,
            description='Argo Workflows CRDs')

        print(
            '✅ Argo Workflows CRDs deployed successfully from local manifests')

    def deploy_pypi_server(self):
        """Deploy PyPI server and upload packages."""
        if not self.args.deploy_pypi_server:
            return

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', 'test-pypiserver'], check=False)

        pypi_resources_path = os.path.join(self.operator_repo_path, '.github',
                                           'resources', 'pypiserver', 'base')

        success = self.deployment_manager.deploy_and_wait(
            manifest_path=pypi_resources_path,
            namespace='test-pypiserver',
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='pypi-server',
            wait_timeout='60s',
            description='PyPI server')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace='test-pypiserver',
                deployment_name='pypi-server',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('PyPI server deployment failed')

        # Apply TLS configuration
        print('🔐 Applying TLS configuration for PyPI server...')
        nginx_tls_config_path = os.path.join(self.operator_repo_path, '.github',
                                             'resources', 'pypiserver', 'base',
                                             'nginx-tls-config.yaml')

        for namespace in ['test-pypiserver', self.deployment_namespace]:
            print(f'🔗 Applying TLS config to namespace: {namespace}')
            self.deployment_manager.apply_resource(
                manifest_path=nginx_tls_config_path,
                namespace=namespace,
                description=f'TLS config for {namespace}')

        # Upload packages
        print('📦 Uploading Python packages to PyPI server...')
        upload_script_path = os.path.join(self.operator_repo_path, '.github',
                                          'scripts', 'python_package_upload')

        self.deployment_manager.run_command(['bash', 'package_upload_run.sh'],
                                            cwd=upload_script_path)

        print('✅ PyPI server deployed and packages uploaded successfully')

    def setup_service_account_rbac(self, is_operator: bool):
        """Set up RBAC permissions for the deployment service account."""
        print('🔐 Setting up RBAC permissions for service account...')

        if is_operator:
            service_account_name = f'ds-pipeline-{self.dspa_name}'
        else:
            service_account_name = 'ml-pipeline'

        print(f'🔧 Creating RBAC for service account: {service_account_name}')

        cluster_role_manifest = {
            'apiVersion': 'rbac.authorization.k8s.io/v1',
            'kind': 'ClusterRole',
            'metadata': {
                'name': f'{service_account_name}-pipeline-role'
            },
            'rules': [
                {
                    'apiGroups': ['*'],
                    'resources': ['*'],
                    'verbs': ['create', 'get', 'list', 'watch', 'update', 'patch', 'delete']
                }
            ]
        }

        cluster_role_binding_manifest = {
            'apiVersion': 'rbac.authorization.k8s.io/v1',
            'kind': 'ClusterRoleBinding',
            'metadata': {
                'name': f'{service_account_name}-pipeline-binding'
            },
            'subjects': [
                {
                    'kind': 'ServiceAccount',
                    'name': service_account_name,
                    'namespace': self.deployment_namespace
                }
            ],
            'roleRef': {
                'kind': 'ClusterRole',
                'name': f'{service_account_name}-pipeline-role',
                'apiGroup': 'rbac.authorization.k8s.io'
            }
        }

        cluster_role_file = os.path.join(self.temp_dir, 'cluster-role.yaml')
        with open(cluster_role_file, 'w') as f:
            yaml.dump(cluster_role_manifest, f, default_flow_style=False)

        self.deployment_manager.apply_resource(
            manifest_path=cluster_role_file,
            description=f'ClusterRole for {service_account_name}')

        cluster_role_binding_file = os.path.join(self.temp_dir,
                                                 'cluster-role-binding.yaml')
        with open(cluster_role_binding_file, 'w') as f:
            yaml.dump(cluster_role_binding_manifest, f,
                      default_flow_style=False)

        self.deployment_manager.apply_resource(
            manifest_path=cluster_role_binding_file,
            description=f'ClusterRoleBinding for {service_account_name}')

        print(
            f'✅ RBAC permissions configured for service account: {service_account_name}'
        )

    def forward_port(self, is_operator: bool):
        """Forward API server port to localhost."""
        if not self.args.forward_port:
            return

        print('🔗 Setting up port forwarding...')

        if is_operator:
            api_server_app_name = f'ds-pipeline-{self.dspa_name}'
        else:
            api_server_app_name = 'ml-pipeline'

        print(f'🔍 Looking for pods with app label: {api_server_app_name}')

        pods_result = self.deployment_manager.run_command([
            'kubectl', 'get', 'pods', '-n', self.deployment_namespace, '-l',
            f'app={api_server_app_name}',
            '--field-selector=status.phase=Running', '--no-headers', '-o',
            'custom-columns=NAME:.metadata.name'
        ], check=False)

        if pods_result.returncode != 0 or not pods_result.stdout.strip():
            print(
                f'❌ No running pods found with label app={api_server_app_name} in namespace {self.deployment_namespace}'
            )
            print('🔍 Available pods in namespace:')
            self.deployment_manager.run_command([
                'kubectl', 'get', 'pods', '-n', self.deployment_namespace, '-o',
                'wide'
            ], check=False)
            raise RuntimeError(
                f'Port forwarding failed: No running API server pods found with label app={api_server_app_name}'
            )

        pod_names = [
            name.strip()
            for name in pods_result.stdout.strip().split('\n')
            if name.strip()
        ]
        print(
            f"✅ Found {len(pod_names)} running pod(s): {', '.join(pod_names)}")

        forward_script = './.github/resources/scripts/forward-port.sh'
        try:
            self.deployment_manager.run_command([
                'bash', forward_script, '-q', self.deployment_namespace,
                api_server_app_name, '8888', '8888'
            ])
            print('✅ Port forwarding setup completed')
        except subprocess.CalledProcessError as e:
            print(f'❌ Port forwarding failed with exit code {e.returncode}')
            if e.output:
                print(f'❌ Error output: {e.output}')
            raise RuntimeError(f'Port forwarding setup failed: {str(e)}')
