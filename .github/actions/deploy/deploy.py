#!/usr/bin/env python3
"""Enhanced Data Science Pipelines Deployment Script.

This script provides flexible deployment options for:
1. Data Science Pipelines Operator (DSPO)
2. PyPI Server for Python packages
3. Data Science Pipelines (either via DSPO CR or direct manifests)

Note: Assumes Docker images are already built and available in the registry
via the build action.
"""

import argparse
import os
import subprocess
import tempfile
from typing import Any, Dict, List

import yaml


class DSPDeployer:

    def __init__(self, args):
        self.args = args
        self.repo_owner = None
        self.target_branch = None
        self.operator_repo_path = None
        self.temp_dir = None
        self.deployment_namespace = None  # Will be set based on deployment mode
        self.dspa_name = 'dspa-test'  # DSPA resource name
        self.operator_namespace = None
        self.operator_deployment = 'data-science-pipelines-operator-controller-manager'

        # Convert string arguments to booleans once
        self._convert_args_to_booleans()

    def str_to_bool(self, value: str) -> bool:
        """Convert string values to boolean."""
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            return value.lower() == 'true'
        return bool(value)

    def _convert_args_to_booleans(self):
        """Convert string arguments to boolean values."""
        boolean_args = [
            'deploy_pypi_server', 'deploy_external_argo', 'proxy',
            'cache_enabled', 'multi_user', 'artifact_proxy', 'forward_port',
            'pod_to_pod_tls_enabled', 'deploy_external_db'
        ]

        for arg_name in boolean_args:
            if hasattr(self.args, arg_name):
                current_value = getattr(self.args, arg_name)
                setattr(self.args, arg_name, self.str_to_bool(current_value))

    def setup_environment(self):
        """Extract repository information and set up environment."""
        print('🔧 Setting up deployment environment...')

        # Extract repo owner from github repository
        if self.args.github_repository:
            self.repo_owner = self.args.github_repository.split('/')[0]
            print(f'📂 Detected repository owner: {self.repo_owner}')
        else:
            raise ValueError('GitHub repository not provided')

        # Set target branch
        self.target_branch = self.args.github_base_ref or 'main'
        print(f'🌳 Target branch: {self.target_branch}')

        # Determine operator namespace
        if self.repo_owner == 'red-hat-data-services':
            self.operator_namespace = 'rhods'
        else:
            self.operator_namespace = 'opendatahub'

        # Create temp directory for operations
        self.temp_dir = tempfile.mkdtemp()
        print(f'📁 Working directory: {self.temp_dir}')

        # Set deployment namespace from input args
        self.deployment_namespace = self.args.namespace
        print(f'🏷️  Deployment namespace: {self.deployment_namespace}')

        # Create deployment namespace early since it's needed for secrets
        print(f'🏷️  Creating deployment namespace: {self.deployment_namespace}')
        self.run_command(
            ['kubectl', 'create', 'namespace', self.deployment_namespace],
            check=False  # Don't fail if namespace already exists
        )

    def run_command(self,
                    cmd: List[str],
                    cwd: str = None,
                    check: bool = True,
                    timeout: int = None,
                    env: dict = None) -> subprocess.CompletedProcess:
        """Run shell command with streaming output to avoid memory issues."""
        cmd_str = ' '.join(cmd)
        print(f'🚀 Running: {cmd_str}')
        if cwd:
            print(f'📂 Working directory: {cwd}')
        if timeout:
            print(f'⏱️  Timeout: {timeout} seconds')

        process = None
        try:
            # Use Popen for streaming output instead of run() to avoid memory issues
            process = subprocess.Popen(
                cmd,
                cwd=cwd,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,  # Merge stderr into stdout
                text=True,
                bufsize=1,  # Line buffered
                universal_newlines=True,
                env=env)

            # Iterate over the process's stdout line by line in real time
            output_lines = []
            for line in process.stdout:
                print(
                    line, end='',
                    flush=True)  # Use flush=True to ensure immediate printing
                output_lines.append(line.rstrip())

            # Wait for the process to finish and get the return code
            return_code = process.wait(timeout=timeout)

            # Create a mock CompletedProcess for compatibility
            result = subprocess.CompletedProcess(
                args=cmd,
                returncode=return_code,
                stdout='\n'.join(output_lines),
                stderr=''  # Already merged into stdout
            )

            if check and return_code != 0:
                raise subprocess.CalledProcessError(
                    return_code, cmd, output=result.stdout)

            return result

        except subprocess.TimeoutExpired as e:
            print(f'⏰ Command timed out after {timeout} seconds')
            print(f'❌ Timeout command: {cmd_str}')
            raise
        except subprocess.CalledProcessError as e:
            # Error details already streamed during execution
            print(f'❌ Command failed with exit code {e.returncode}')
            raise
        except Exception as e:
            print(f'❌ Unexpected error running command: {e}')
            if process is not None:
                if process.poll() is None:  # Check if process is still running
                    process.kill()
                process.wait()
            raise

    def clone_operator_repo(self) -> str:
        """Clone data-science-pipelines-operator repository."""
        operator_repo_url = f'https://github.com/{self.repo_owner}/data-science-pipelines-operator'
        operator_path = os.path.join(self.temp_dir,
                                     'data-science-pipelines-operator')

        print(f'📥 Cloning operator repository: {operator_repo_url}')
        self.run_command(['git', 'clone', operator_repo_url, operator_path])

        # Map target branch to operator branch (master -> main for operator repo)
        operator_branch = 'main' if self.target_branch == 'master' else self.target_branch

        print(f'🔄 Checking out branch: {operator_branch}')
        try:
            self.run_command(['git', 'checkout', operator_branch],
                             cwd=operator_path)
        except subprocess.CalledProcessError:
            print(
                f'⚠️  Branch {operator_branch} not found, using default branch')

        # Fix Makefile permissions if it exists
        makefile_path = os.path.join(operator_path, 'Makefile')
        if os.path.exists(makefile_path):
            print('🔧 Fixing Makefile permissions...')
            self.run_command(['chmod', '644', makefile_path],
                             cwd=operator_path,
                             check=False)

        self.operator_repo_path = operator_path
        return operator_path

    def needs_operator_repo(self) -> bool:
        """Check if we need to clone the operator repository."""
        return self.args.deploy_pypi_server

    def deploy_operator(self):
        """Deploy Data Science Pipelines Operator."""
        print('🔧 Deploying Data Science Pipelines Operator...')

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        # Determine dspo branch based on target branch
        dspo_branch = 'main' if self.target_branch == 'master' else self.target_branch

        # Determine repo based on repo owner
        repo = 'opendatahub' if self.repo_owner == 'opendatahub-io' else 'rhoai'

        operator_image = f'quay.io/{repo}/data-science-pipelines-operator:{dspo_branch}'

        print(f'🏷️  Using operator image: {operator_image}')

        # Create operator namespace if it doesn't exist
        print(f'🏷️  Creating operator namespace: {self.operator_namespace}')
        self.run_command(
            ['kubectl', 'create', 'namespace', self.operator_namespace],
            check=False  # Don't fail if namespace already exists
        )

        # Install CRDs first to avoid ServiceMonitor errors
        print('🔧 Installing operator CRDs...')

        # 1. Install standard CRDs via make
        self.run_command(['make', 'install'], cwd=self.operator_repo_path)

        # 2. Apply additional CRDs from resources directory (like tests.sh)
        print('🔧 Installing additional CRDs from resources directory...')
        additional_crds_path = os.path.join(self.operator_repo_path, '.github',
                                            'resources', 'crds')
        if os.path.exists(additional_crds_path):
            self.run_command(['kubectl', 'apply', '-f', additional_crds_path],
                             check=True)  # Required for ServiceMonitor CRD

        # 3. Apply external route CRD (OpenShift specific, like tests.sh)
        print('🔧 Installing OpenShift route CRD...')
        route_crd_path = os.path.join(self.operator_repo_path, 'config', 'crd',
                                      'external',
                                      'route.openshift.io_routes.yaml')
        if os.path.exists(route_crd_path):
            self.run_command(['kubectl', 'apply', '-f', route_crd_path],
                             check=False)

        # Deploy using make with specified operator image
        # Set IMAGES_DSPO environment variable that the Makefile expects
        deploy_env = {'IMAGES_DSPO': operator_image, 'IMG': operator_image}

        # Add current environment variables
        deploy_env.update(os.environ)

        print(f'🔧 Setting IMAGES_DSPO={operator_image}')
        self.run_command(['make', 'deploy-kind', f'IMG={operator_image}'],
                         cwd=self.operator_repo_path,
                         env=deploy_env)

        # Debug ConfigMap creation (like tests.sh dependency verification)
        print('🔍 Checking created ConfigMaps...')
        self.run_command(
            ['kubectl', 'get', 'configmaps', '-n', self.operator_namespace],
            check=False)

        # Verify ConfigMap creation (like tests.sh wait_for_dspo_dependencies)
        print('🔧 Verifying DSPO ConfigMap creation...')
        configmap_names = [
            'data-science-pipelines-operator-dspo-config',
            'dspo-config'  # Fallback in case the name doesn't have prefix
        ]

        configmap_found = False
        for cm_name in configmap_names:
            result = self.run_command([
                'kubectl', 'get', 'configmap', cm_name, '-n',
                self.operator_namespace
            ],
                                      check=False)

            if result.returncode == 0:
                print(f'✅ Found required ConfigMap: {cm_name}')
                configmap_found = True
                break

        if not configmap_found:
            print(f'⚠️  Required ConfigMaps not found. Available ConfigMaps:')
            self.run_command([
                'kubectl', 'get', 'configmaps', '-n', self.operator_namespace,
                '--no-headers', '-o', 'custom-columns=NAME:.metadata.name'
            ],
                             check=False)

        # Wait for operator to be ready

        print(
            f'⏳ Waiting for operator to be ready in namespace: {self.operator_namespace}...'
        )
        result = self.run_command([
            'kubectl', 'wait', '--for=condition=Available=true', 'deployment',
            '--all', '-n', self.operator_namespace, '--timeout=300s'
        ],
                                  check=False)

        if result.returncode != 0:
            print(
                f'⚠️  Operator did not become ready within timeout, investigating...'
            )

            # Get deployment status for debugging
            print('🔍 Checking deployment status...')
            self.run_command([
                'kubectl', 'get', 'deployments', '-n', self.operator_namespace
            ],
                             check=False)

            # Get pod status for debugging
            print('🔍 Checking pod status...')
            self.run_command([
                'kubectl', 'get', 'pods', '-n', self.operator_namespace, '-o',
                'wide'
            ],
                             check=False)

            # Find and describe specific operator-manager pods
            print('🔍 Finding operator-manager pods...')
            pod_result = self.run_command([
                'kubectl', 'get', 'pods', '-n', self.operator_namespace, '-l',
                'app.kubernetes.io/name=data-science-pipelines-operator',
                '--no-headers', '-o', 'custom-columns=NAME:.metadata.name'
            ],
                                          check=False)

            if pod_result.returncode == 0 and pod_result.stdout.strip():
                pod_names = [
                    name.strip()
                    for name in pod_result.stdout.strip().split('\n')
                    if name.strip()
                ]

                for pod_name in pod_names:
                    print(f'🔍 Describing pod: {pod_name}')
                    self.run_command([
                        'kubectl', 'describe', 'pod', pod_name, '-n',
                        self.operator_namespace
                    ],
                                     check=False)

                    print(f'🔍 Getting events for pod: {pod_name}')
                    self.run_command([
                        'kubectl', 'get', 'events', '-n',
                        self.operator_namespace, '--field-selector',
                        f'involvedObject.name={pod_name}',
                        '--sort-by=.lastTimestamp'
                    ],
                                     check=False)

                    print(f'🔍 Checking pod logs: {pod_name}')
                    self.run_command([
                        'kubectl', 'logs', pod_name, '-n',
                        self.operator_namespace, '--tail=50'
                    ],
                                     check=False)
            else:
                print('⚠️  No operator-manager pods found with expected labels')

            # Get all events in the namespace for broader context
            print('🔍 Getting all recent events in operator namespace...')
            self.run_command([
                'kubectl', 'get', 'events', '-n', self.operator_namespace,
                '--sort-by=.lastTimestamp', '--limit=20'
            ],
                             check=False)

            error_msg = f'Operator did not become ready within timeout. kubectl wait failed with exit code {result.returncode}'
            if result.stderr:
                error_msg += f'. Error: {result.stderr.strip()}'
            print(f'❌ {error_msg}')
            raise RuntimeError(error_msg)

        # Configure operator for external Argo if requested
        if self.args.deploy_external_argo:
            self._configure_operator_for_external_argo(self.operator_namespace)

        print('✅ Data Science Pipelines Operator deployed successfully')

    def deploy_pypi_server(self):
        """Deploy PyPI server using operator repository resources and upload
        packages."""
        if not self.args.deploy_pypi_server:
            return

        print('🐍 Deploying PyPI server and uploading packages...')

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        # Create namespace
        self.run_command(['kubectl', 'create', 'namespace', 'test-pypiserver'],
                         check=False)

        # Deploy PyPI server
        pypi_resources_path = os.path.join(self.operator_repo_path, '.github',
                                           'resources', 'pypiserver', 'base')

        self.run_command([
            'kubectl', '-n', 'test-pypiserver', 'apply', '-k',
            pypi_resources_path
        ])

        # Wait for PyPI server to be ready
        print('⏳ Waiting for PyPI server to be ready...')
        self.run_command([
            'kubectl', 'wait', '-n', 'test-pypiserver', '--timeout=60s',
            '--for=condition=Available=true', 'deployment', 'pypi-server'
        ])

        # Apply TLS configuration to relevant namespaces
        print('🔐 Applying TLS configuration for PyPI server...')
        nginx_tls_config_path = os.path.join(self.operator_repo_path, '.github',
                                             'resources', 'pypiserver', 'base',
                                             'nginx-tls-config.yaml')

        # Apply to both PyPI server namespace and deployment namespace
        for namespace in ['test-pypiserver', self.deployment_namespace]:
            print(f'🔗 Applying TLS config to namespace: {namespace}')
            self.run_command([
                'kubectl', 'apply', '-f', nginx_tls_config_path, '-n', namespace
            ],
                             check=False)  # Don't fail if config doesn't exist

        # Upload Python packages automatically when PyPI server is deployed
        print('📦 Uploading Python packages to PyPI server...')
        upload_script_path = os.path.join(self.operator_repo_path, '.github',
                                          'scripts', 'python_package_upload')

        self.run_command(['bash', 'package_upload_run.sh'],
                         cwd=upload_script_path)

        print('✅ PyPI server deployed and packages uploaded successfully')

    def deploy_seaweedfs(self):
        """Deploy SeaweedFS using local manifests."""
        if self.args.storage_backend != 'seaweedfs':
            return

        print('🌊 Deploying SeaweedFS using local manifests...')

        # Use local SeaweedFS manifests from the repository
        seaweedfs_path = './manifests/kustomize/third-party/seaweedfs/base'

        # Verify the kustomization file exists
        if not os.path.exists(
                os.path.join(seaweedfs_path, 'kustomization.yaml')):
            raise ValueError(
                f'SeaweedFS kustomization.yaml not found at {seaweedfs_path}')

        print(
            f'📦 Applying SeaweedFS manifests from local path: {seaweedfs_path}')

        # Apply SeaweedFS manifests to deployment namespace
        self.run_command([
            'kubectl', '-n', self.deployment_namespace, 'apply', '-k',
            seaweedfs_path
        ])

        # Wait for SeaweedFS to be ready
        print('⏳ Waiting for SeaweedFS to be ready...')
        self.run_command([
            'kubectl', 'wait', '-n', self.deployment_namespace,
            '--timeout=300s', '--for=condition=Available=true', 'deployment',
            'seaweedfs'
        ])

        # Wait for SeaweedFS init job to complete (S3 auth setup)
        print('⏳ Waiting for SeaweedFS init job to complete...')

        # First check if the job exists and debug its status
        print('🔍 Checking SeaweedFS init job status...')
        self.run_command([
            'kubectl', 'get', 'job', 'init-seaweedfs', '-n',
            self.deployment_namespace, '-o', 'yaml'
        ],
                         check=False)

        self.run_command([
            'kubectl', 'get', 'pods', '-n', self.deployment_namespace, '-l',
            'job-name=init-seaweedfs'
        ],
                         check=False)

        # Try to wait for completion with more detailed error handling
        result = self.run_command([
            'kubectl', 'wait', '-n', self.deployment_namespace,
            '--timeout=300s', '--for=condition=complete', 'job',
            'init-seaweedfs'
        ],
                                  check=False)

        if result.returncode != 0:
            print(
                '⚠️  Init job did not complete within timeout, checking logs...'
            )
            self.run_command([
                'kubectl', 'logs', '-n', self.deployment_namespace, '-l',
                'job-name=init-seaweedfs', '--tail=50'
            ],
                             check=False)
            print('⚠️  Continuing without waiting for init job completion...')

        print('✅ SeaweedFS deployed successfully from local manifests')

    def deploy_cert_manager(self):
        """Deploy cert-manager for certificate management."""
        print('🔐 Deploying cert-manager...')

        cert_manager_namespace = 'cert-manager'

        # Create cert-manager namespace
        print(f'🏷️  Creating cert-manager namespace: {cert_manager_namespace}')
        self.run_command(
            ['kubectl', 'create', 'namespace', cert_manager_namespace],
            check=False  # Don't fail if namespace already exists
        )

        # Deploy cert-manager (same URL as upstream tests.sh)
        print('📜 Deploying cert-manager...')
        self.run_command([
            'kubectl', 'apply', '-f',
            'https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml'
        ])

        # Wait for cert-manager to be ready (same timeout as upstream)
        print('⏳ Waiting for cert-manager pods to be ready...')
        self.run_command([
            'kubectl', 'wait', '-n', cert_manager_namespace, '--timeout=90s',
            '--for=condition=Ready', 'pods', '--all'
        ])

        print('✅ Cert-manager deployed successfully')

    def apply_webhook_certs(self):
        """Apply webhook certificates for TLS communication."""
        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for webhook certificates')

        print('📜 Applying webhook certificates for TLS communication...')

        webhook_certs_path = os.path.join(self.operator_repo_path, '.github',
                                          'resources', 'webhook')

        if os.path.exists(webhook_certs_path):
            self.run_command([
                'kubectl', '-n', self.operator_namespace, 'apply', '-k',
                webhook_certs_path
            ])
            print('✅ Webhook certificates applied for TLS communication')
        else:
            print(
                f'⚠️  Webhook certificates path not found: {webhook_certs_path}'
            )

    def deploy_external_argo(self):
        """Deploy Argo Workflows externally using local manifests."""
        if not self.args.deploy_external_argo:
            return

        print(
            '⚙️  Deploying Argo Workflows externally using local manifests...')

        argo_version = self.args.argo_version or 'v3.6.7'

        # Update Argo version if specified
        if argo_version:
            print(
                f'📝 NOTE: Argo version {argo_version} specified, updating Argo Workflow manifests...'
            )

            # Write version to VERSION file
            version_file = './manifests/kustomize/third-party/argo/VERSION'
            with open(version_file, 'w') as f:
                f.write(argo_version + '\n')
            print(f'📄 Written {argo_version} to {version_file}')

            # Update manifests using make
            print('🔄 Updating Argo manifests...')
            self.run_command([
                'make', '-C', './manifests/kustomize/third-party/argo', 'update'
            ])
            print(f'✅ Manifests updated for Argo version {argo_version}')

        # Apply CRDs from local manifests
        print('📦 Applying Argo CRDs from local manifests...')
        crds_path = './manifests/kustomize/third-party/argo/installs/namespace/cluster-scoped'

        self.run_command(['kubectl', 'apply', '-k', crds_path])

        print(
            '✅ Argo Workflows CRDs deployed successfully from local manifests')

    def _configure_operator_for_external_argo(self, operator_namespace: str):
        """Configure the deployed operator to use external Argo Workflows."""
        print('🔧 Configuring operator to use external Argo Workflows...')

        # Patch the operator deployment to set DSPO_ARGOWORKFLOWSCONTROLLERS environment variable
        # This tells the operator to not deploy its own Argo and use external one instead
        patch_json = '[{"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "DSPO_ARGOWORKFLOWSCONTROLLERS", "value": "{\\"managementState\\": \\"Removed\\"}"}}]'

        self.run_command([
            'kubectl', 'patch', 'deployment', self.operator_deployment, '-n',
            operator_namespace, '--type=json', '-p', patch_json
        ])

        # Wait for the deployment to roll out with new configuration
        print(
            '⏳ Waiting for operator to restart with external Argo configuration...'
        )
        self.run_command([
            'kubectl', 'rollout', 'status',
            f'deployment/{self.operator_deployment}', '-n', operator_namespace,
            '--timeout=300s'
        ])

        print('✅ Operator configured for external Argo successfully')

    def generate_dspa_yaml(self) -> Dict[str, Any]:
        """Generate DataSciencePipelinesApplication YAML."""
        print('📄 Generating DSPA configuration...')

        # Configure API server with proper fields (not environment variables)
        api_server_config = {
            'image':
                f'{self.args.image_registry}/apiserver:{self.args.image_tag}',
            'argoDriverImage':
                f'{self.args.image_registry}/driver:{self.args.image_tag}',
            'argoLauncherImage':
                f'{self.args.image_registry}/launcher:{self.args.image_tag}',
            'cacheEnabled':
                self.args.cache_enabled,
            'enableOauth':
                False  # Disable OAuth to avoid TLS proxy issues
        }

        # Add CA bundle configuration when TLS is enabled
        if self.args.pod_to_pod_tls_enabled:
            api_server_config.update({
                'caBundleFileName': 'service-ca.crt',
                'cABundle': {
                    'configMapName': 'openshift-service-ca.crt',
                    'configMapKey': 'service-ca.crt'
                }
            })
            print('🔧 Added CA bundle configuration for TLS communication')

        # Add Kubernetes native mode if specified
        if self.args.pipeline_store == 'kubernetes':
            api_server_config['pipelineStore'] = 'kubernetes'
            print('🔧 Enabling Kubernetes native pipeline storage')

        # Use images from registry (built by build action)
        dspa_config = {
            'apiVersion': 'datasciencepipelinesapplications.opendatahub.io/v1',
            'kind': 'DataSciencePipelinesApplication',
            'metadata': {
                'name': self.dspa_name,
                'namespace': self.deployment_namespace
            },
            'spec': {
                'dspVersion': 'v2',
                'apiServer': api_server_config,
                'persistenceAgent': {
                    'image':
                        f'{self.args.image_registry}/persistenceagent:{self.args.image_tag}'
                },
                'scheduledWorkflow': {
                    'image':
                        f'{self.args.image_registry}/scheduledworkflow:{self.args.image_tag}'
                },
                'podToPodTLS': self.args.pod_to_pod_tls_enabled
            }
        }

        # Add storage configuration
        if self.args.storage_backend == 'minio':
            dspa_config['spec']['objectStorage'] = {
                'enableExternalRoute': True,
                'minio': {
                    'deploy':
                        True,
                    'image':
                        'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
                }
            }
        else:  # seaweedfs (default)
            dspa_config['spec']['objectStorage'] = {
                'externalStorage': {
                    'host':
                        f'seaweedfs.{self.deployment_namespace}.svc.cluster.local',
                    'port':
                        '8333',
                    'bucket':
                        'mlpipeline',
                    'region':
                        'us-east-1',  # Required but not used by SeaweedFS
                    'scheme':
                        'http',
                    's3CredentialsSecret': {
                        'accessKey': 'accesskey',
                        'secretKey': 'secretkey',
                        'secretName': 'mlpipeline-minio-artifact'
                    }
                }
            }

        # Add database configuration with MariaDB image override
        dspa_config['spec']['database'] = {
            'mariaDB': {
                'image': 'quay.io/sclorg/mariadb-105-c9s:latest'
            }
        }

        return dspa_config

    def deploy_dsp_via_operator(self):
        """Deploy Data Science Pipelines via operator using DSPA CR."""
        print('🚀 Deploying Data Science Pipelines via operator...')

        # Generate DSPA configuration
        dspa_config = self.generate_dspa_yaml()

        # Write DSPA to file
        dspa_file = os.path.join(self.temp_dir, 'dspa.yaml')
        with open(dspa_file, 'w') as f:
            yaml.dump(dspa_config, f, default_flow_style=False)

        print(f'📝 DSPA configuration written to: {dspa_file}')
        print(
            f'📄 DSPA Content:\n{yaml.dump(dspa_config, default_flow_style=False)}'
        )

        # Create namespace if it doesn't exist
        self.run_command(
            ['kubectl', 'create', 'namespace', self.deployment_namespace],
            check=False)

        # Create OpenShift service CA ConfigMap for compatibility (when TLS enabled)
        if self.args.pod_to_pod_tls_enabled:
            self._create_openshift_service_ca_configmap()

        # Create DSPA deployment
        print('Creating DSPA deployment')
        self.run_command([
            'kubectl', 'apply', '-n', self.deployment_namespace, '-f', dspa_file
        ])

        deployment_name = f'ds-pipeline-{self.dspa_name}'
        deployment_wait_time = '2'

        # Wait for deployment to exist with timeout
        print(f'⏳ Waiting for deployment {deployment_name} to be created...')
        wait_cmd = [
            'timeout', f'{deployment_wait_time}m', 'bash', '-c',
            f'until kubectl -n {self.deployment_namespace} get deployment {deployment_name} &> /dev/null; do echo "Waiting for the deployment {deployment_name}..."; sleep 10; done'
        ]

        result = self.run_command(wait_cmd, check=False)
        if result.returncode != 0:
            print(
                f'❌ Deployment {deployment_name} was not created within {deployment_wait_time} minute timeout'
            )
            print(f'Investigating Deployment: {deployment_name}')
            self._investigate_deployment_failure(self.deployment_namespace,
                                                 deployment_name)
            print(f'Investigating Deployment: {self.operator_deployment}')
            self._investigate_deployment_failure(self.operator_namespace,
                                                 self.operator_deployment)
            raise RuntimeError(
                f'Deployment {deployment_name} was not created within timeout')

        # Wait for deployment to be available
        print(f'⏳ Waiting for deployment {deployment_name} to be available...')
        wait_result = self.run_command([
            'kubectl', 'wait', '--for=condition=available',
            f'deployment/ds-pipeline-{self.dspa_name}', '--timeout=10m', '-n',
            self.deployment_namespace
        ],
                                       check=False)

        if wait_result.returncode == 0:
            print('Operator pod is ready')
            print('✅ Data Science Pipelines deployed via operator successfully')
        else:
            print(
                'Warning: Operator pod did not become ready within timeout, continuing anyway...'
            )
            self._investigate_deployment_failure(self.deployment_namespace,
                                                 deployment_name)
            raise RuntimeError('DSPA did not become ready within timeout')

    def _investigate_deployment_failure(self,
                                        namespace: str,
                                        deployment_name: str = None):
        """Get pods and print logs of failed pods when deployment times out.

        Args:
            namespace: The namespace to investigate
            deployment_name: Optional specific deployment name for context
        """
        print(f'🔍 Investigating deployment failure in namespace: {namespace}')
        if deployment_name:
            print(f'🔍 Context: Deployment {deployment_name} failed')

        # Get all pods in the namespace
        print('🔍 All pods in namespace:')
        self.run_command(
            ['kubectl', 'get', 'pods', '-n', namespace, '-o', 'wide'],
            check=False)

        # Get all pods first
        print('🔍 Getting all pods in namespace...')
        pod_result = self.run_command([
            'kubectl', 'get', 'pods', '-n', namespace, '--no-headers', '-o',
            'custom-columns=NAME:.metadata.name,STATUS:.status.phase'
        ],
                                      check=False)

        if pod_result.returncode == 0 and pod_result.stdout.strip():
            all_pods = []
            failed_pods = []
            running_pods = []

            for line in pod_result.stdout.strip().split('\n'):
                if line.strip():
                    parts = line.strip().split()
                    if len(parts) >= 2:
                        pod_name = parts[0]
                        status = parts[1]
                        all_pods.append((pod_name, status))

                        # Separate failed/pending vs running/succeeded pods
                        if status not in ['Running', 'Succeeded']:
                            failed_pods.append((pod_name, status))
                        elif status == 'Running':
                            running_pods.append((pod_name, status))

            # Process failed/pending pods if any exist
            if failed_pods:
                print(f'🔍 Found {len(failed_pods)} failed/pending pods')
                for pod_name, status in failed_pods:
                    print(f'🔍 Investigating {status} pod: {pod_name}')

                    # Describe the pod
                    print(f'🔍 Describing pod: {pod_name}')
                    self.run_command([
                        'kubectl', 'describe', 'pod', pod_name, '-n', namespace
                    ],
                                     check=False)

                    # Get pod logs (current)
                    print(f'🔍 Current logs from pod: {pod_name}')
                    self.run_command([
                        'kubectl', 'logs', pod_name, '-n', namespace,
                        '--tail=100'
                    ],
                                     check=False)

                    # Get pod logs (previous if available)
                    print(f'🔍 Previous logs from pod: {pod_name} (if any)')
                    self.run_command([
                        'kubectl', 'logs', pod_name, '-n', namespace,
                        '--previous', '--tail=50'
                    ],
                                     check=False)
            else:
                print('🔍 No failed/pending pods found')

                # Get last 30 log lines from all running pods
                if running_pods:
                    print(
                        f'🔍 Getting last 30 log lines from {len(running_pods)} running pods...'
                    )
                    for pod_name, status in running_pods:
                        print(
                            f'🔍 Last 30 log lines from {status} pod: {pod_name}'
                        )
                        self.run_command([
                            'kubectl', 'logs', pod_name, '-n', namespace,
                            '--tail=30'
                        ],
                                         check=False)
                else:
                    print('🔍 No running pods found to collect logs from')
        else:
            print('🔍 No pods found in namespace')

        # Get recent events in namespace
        print('🔍 Recent events in namespace:')
        self.run_command([
            'kubectl', 'get', 'events', '-n', namespace,
            '--sort-by=.lastTimestamp', '--limit=30'
        ],
                         check=False)

    def deploy_dsp_direct(self):
        """Deploy Data Science Pipelines using direct manifests (existing
        logic)"""
        print('🚀 Deploying Data Science Pipelines using direct manifests...')

        # Configure deployment arguments
        deploy_args = []

        if self.args.proxy:
            deploy_args.append('--proxy')

        if not self.args.cache_enabled:
            deploy_args.append('--cache-disabled')

        if self.args.pipeline_store == 'kubernetes':
            deploy_args.append('--deploy-k8s-native')

        if self.args.multi_user:
            deploy_args.append('--multi-user')

        if self.args.artifact_proxy:
            deploy_args.append('--artifact-proxy')

        if self.args.storage_backend and self.args.storage_backend != 'seaweedfs':
            deploy_args.extend(['--storage', self.args.storage_backend])

        if self.args.argo_version:
            deploy_args.extend(['--argo-version', self.args.argo_version])

        if self.args.pod_to_pod_tls_enabled:
            deploy_args.append('--tls-enabled')

        # Set up environment with correct REGISTRY variable
        deploy_env = os.environ.copy()
        deploy_env['REGISTRY'] = self.args.image_registry

        print(f'🔧 Setting REGISTRY={self.args.image_registry}')
        print(f'🏷️  Using image tag: {self.args.image_tag}')

        # Call existing deploy script
        deploy_script = './.github/resources/scripts/deploy-kfp.sh'
        cmd = ['bash', deploy_script] + deploy_args

        # Add timeout to prevent hanging on log collection
        self.run_command(cmd, timeout=1800, env=deploy_env)  # 30 minute timeout

        print('✅ Data Science Pipelines deployed directly successfully')

    def forward_port(self):
        """Forward API server port to localhost."""
        if not self.args.forward_port:
            return

        print('🔗 Setting up port forwarding...')

        forward_script = './.github/resources/scripts/forward-port.sh'
        self.run_command([
            'bash', forward_script, '-q', self.deployment_namespace,
            'ml-pipeline', '8888', '8888'
        ])

        print('✅ Port forwarding setup completed')

    def deploy(self):
        """Main deployment orchestration with intelligent mode selection."""
        try:
            self.setup_environment()

            # 🧠 Intelligent deployment mode selection
            use_operator_deployment = self._should_use_operator_deployment()

            if use_operator_deployment:
                print('🔧 Using DSPO (operator) deployment mode')
                # For operator deployment, we always need the operator repo
                self.clone_operator_repo()

                # Deploy cert-manager (always deployed for operator mode)
                self.deploy_cert_manager()

                # Deploy external Argo if requested (must be done before operator)
                self.deploy_external_argo()

                # Deploy operator
                self.deploy_operator()

                # Apply webhook certificates for TLS communication
                self.apply_webhook_certs()

                # Deploy PyPI server if requested (includes package upload)
                self.deploy_pypi_server()

                # Deploy SeaweedFS if using seaweedfs storage (like tests.sh approach)
                self.deploy_seaweedfs()

                # Create operator-expected certificates before deploying DSPA
                self._create_operator_expected_certificates()

                # Deploy DSP via operator
                self.deploy_dsp_via_operator()

            else:
                print(
                    '🔧 Using direct manifest deployment mode (multi-user detected)'
                )

                # Check if we need operator repo for PyPI server features only
                if self.args.deploy_pypi_server:
                    self.clone_operator_repo()
                    self.deploy_pypi_server()

                # Deploy DSP directly
                self.deploy_dsp_direct()

            # Setup port forwarding
            self.forward_port()

            print('🎉 Deployment completed successfully!')

        except Exception as e:
            print(f'❌ Deployment failed: {str(e)}')
            raise
        finally:
            # Cleanup temp directory
            if self.temp_dir and os.path.exists(self.temp_dir):
                import shutil
                shutil.rmtree(self.temp_dir)

    def _should_use_operator_deployment(self) -> bool:
        """Determine whether to use DSPO (operator) or direct deployment.

        Logic:
        - Multi-user mode: DSPO doesn't support it → use direct deployment
        - All other cases: Use DSPO deployment (default)
        """
        if self.args.multi_user:
            print(
                "⚠️  Multi-user mode detected: DSPO doesn't support multi-user, using direct deployment"
            )
            return False
        elif self.args.proxy:
            print(
                "⚠️  Proxy mode detected: DSPO doesn't support proxy, using direct deployment"
            )
            return False

        return True

    def _create_openshift_service_ca_configmap(self):
        """Create kfp-api-tls-cert ConfigMap for TLS compatibility.

        Creates a ConfigMap with sample CA certificate that can be
        referenced by workflows and DSPA configurations.
        """
        print('🔐 Creating kfp-api-tls-cert ConfigMap for TLS compatibility...')

        # Path to the ConfigMap file
        configmap_file = './.github/actions/deploy/openshift-service-ca-cert-configmap.yaml'

        try:
            # Apply the ConfigMap to the deployment namespace
            self.run_command([
                'kubectl', 'apply', '-f', configmap_file, '-n',
                self.deployment_namespace
            ])
            print('✅ Created openshift-service-ca ConfigMap successfully')
        except Exception as e:
            print(f'❌ Failed to create openshift-service-ca ConfigMap: {e}')

    def _create_operator_expected_certificates(self):
        """Create certificates with operator-expected names by reading existing
        manifests.

        Reads existing TLS certificate manifests and creates new ones
        with names that the DSPO operator expects for various
        components.
        """
        if not self.args.pod_to_pod_tls_enabled:
            return

        print('🔐 Creating operator-expected certificates...')

        # Path to the base TLS certificates manifest directory
        base_cert_dir = './manifests/kustomize/env/cert-manager/base-tls-certs'
        cert_files = ['kfp-api-cert.yaml', 'kfp-api-cert-issuer.yaml']

        temp_cert_files = []

        try:
            for cert_file in cert_files:
                cert_path = os.path.join(base_cert_dir, cert_file)
                if not os.path.exists(cert_path):
                    print(f'⚠️  Certificate file not found: {cert_path}')
                    continue

                print(f'📄 Reading certificate manifest: {cert_path}')
                with open(cert_path, 'r') as f:
                    cert_documents = list(yaml.safe_load_all(f))

                # Process each document in the YAML file
                for doc in cert_documents:
                    if not doc:
                        continue

                    # Create MariaDB TLS certificate based on the main certificate
                    if doc.get('kind') == 'Certificate' and doc.get(
                            'metadata', {}).get('name') == 'kfp-api-tls-cert':
                        mariadb_cert = self._create_mariadb_certificate(doc)
                        if mariadb_cert:
                            # Write to temporary file
                            temp_file = os.path.join(
                                self.temp_dir,
                                f'mariadb-tls-cert-{self.dspa_name}.yaml')
                            with open(temp_file, 'w') as f:
                                yaml.dump(
                                    mariadb_cert, f, default_flow_style=False)
                            temp_cert_files.append(temp_file)
                            print(
                                f'📝 Created MariaDB certificate manifest: {temp_file}'
                            )

                    # Apply issuer if found
                    elif doc.get('kind') == 'Issuer':
                        temp_file = os.path.join(
                            self.temp_dir, f'cert-issuer-{self.dspa_name}.yaml')
                        with open(temp_file, 'w') as f:
                            yaml.dump(doc, f, default_flow_style=False)
                        temp_cert_files.append(temp_file)
                        print(
                            f'📝 Created certificate issuer manifest: {temp_file}'
                        )

            # Apply all certificate manifests
            for temp_file in temp_cert_files:
                print(f'🚀 Applying certificate manifest: {temp_file}')
                self.run_command([
                    'kubectl', 'apply', '-f', temp_file, '-n',
                    self.deployment_namespace
                ])

            print('✅ Operator-expected certificates created successfully')

        except Exception as e:
            print(f'❌ Failed to create operator-expected certificates: {e}')
            raise

    def _create_mariadb_certificate(
            self, base_cert: Dict[str, Any]) -> Dict[str, Any]:
        """Create MariaDB TLS certificate based on the main certificate.

        Args:
            base_cert: The base certificate dictionary to clone

        Returns:
            Dictionary containing the MariaDB certificate manifest
        """
        # Expected secret name format: ds-pipelines-mariadb-tls-{dspa-name}
        mariadb_secret_name = f'ds-pipelines-mariadb-tls-{self.dspa_name}'
        mariadb_cert_name = f'mariadb-tls-cert-{self.dspa_name}'

        print(
            f'🔧 Creating MariaDB certificate with secret name: {mariadb_secret_name}'
        )

        # Clone the base certificate
        mariadb_cert = yaml.safe_load(yaml.dump(base_cert))

        # Update metadata
        mariadb_cert['metadata']['name'] = mariadb_cert_name

        # Update spec for MariaDB-specific settings
        mariadb_cert['spec']['commonName'] = f'ds-pipeline-db-{self.dspa_name}'
        mariadb_cert['spec']['secretName'] = mariadb_secret_name

        # Update DNS names for MariaDB service
        mariadb_cert['spec']['dnsNames'] = [
            f'ds-pipeline-db-{self.dspa_name}',
            f'ds-pipeline-db-{self.dspa_name}.{self.deployment_namespace}',
            f'ds-pipeline-db-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local',
            'localhost'
        ]

        return mariadb_cert


def main():
    parser = argparse.ArgumentParser(
        description='Deploy Data Science Pipelines')

    # GitHub context
    parser.add_argument(
        '--github-repository',
        required=True,
        help='GitHub repository (owner/repo)')
    parser.add_argument(
        '--github-base-ref', help='GitHub base ref (target branch)')

    # Image configuration (images already built by build action)
    parser.add_argument('--image-tag', required=True, help='Image tag')
    parser.add_argument(
        '--image-registry', required=True, help='Image registry')

    # PyPI deployment options (consolidated)
    parser.add_argument(
        '--deploy-pypi-server',
        default='false',
        help='Deploy PyPI server and upload packages')
    parser.add_argument(
        '--deploy-external-argo',
        default='false',
        help='Deploy Argo Workflows externally in separate namespace')

    # Existing KFP options
    parser.add_argument(
        '--pipeline-store',
        default='database',
        choices=['database', 'kubernetes'],
        help='Pipeline store type')
    parser.add_argument('--proxy', default='false', help='Enable proxy')
    parser.add_argument('--cache-enabled', default='true', help='Enable cache')
    parser.add_argument('--multi-user', default='false', help='Multi-user mode')
    parser.add_argument(
        '--artifact-proxy', default='false', help='Enable artifact proxy')
    parser.add_argument(
        '--storage-backend',
        default='seaweedfs',
        choices=['seaweedfs', 'minio'],
        help='Storage backend')
    parser.add_argument('--argo-version', help='Argo version')
    parser.add_argument(
        '--forward-port', default='true', help='Forward API server port')
    parser.add_argument(
        '--pod-to-pod-tls-enabled',
        default='false',
        help='Enable pod-to-pod TLS')
    parser.add_argument(
        '--namespace', default='kubeflow', help='Namespace for DSPA deployment')

    args = parser.parse_args()

    deployer = DSPDeployer(args)
    deployer.deploy()


if __name__ == '__main__':
    main()
