# Data Science Pipelines Test Runner for OpenShift Clusters

This README provides instructions to run Data Science Pipelines tests against an existing OpenShift cluster using the containerized test environment.

## Overview

The test runner deploys and tests Data Science Pipelines Applications (DSPA) on OpenShift clusters. It creates a complete test environment including the operator deployment, DSPA resources, and executes end-to-end tests using the Ginkgo framework.

## Prerequisites

1. **OpenShift Cluster Access**: An existing OpenShift cluster with admin privileges
2. **Kubeconfig**: Valid kubeconfig file for cluster access
3. **Podman**: Container runtime installed on the host system
4. **Storage Access**: Either S3 credentials or use embedded Minio

## Required Environment Variables

### Core Variables (Required)

Set these environment variables either in the container or via .env file:

```bash
# Required - DSPA instance name
export DSPA_NAME="dspa"

# Required - Base namespace (random suffix will be appended)
export NAMESPACE="dspa-test"
```

### Storage Configuration

#### Option 1: S3 Storage (External)
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export BUCKET="your-s3-bucket"
export REGION="us-east-1"
export SECRET_NAME="dashboard-dspa-secret"
export AWS_ENDPOINT="s3.amazonaws.com"  # Optional, defaults from .env
```

#### Option 2: Embedded Minio Storage
```bash
# No additional variables needed - Minio will be deployed automatically
# if AWS credentials are not provided
```

## Command Line Parameters

The test script supports the following parameters:

| Parameter | Description                               | Default Value                                 | Example |
|-----------|-------------------------------------------|-----------------------------------------------|---------|
| `--label-filter` | Ginkgo label filter for test selection    | `"E2ECritical"`                               | `--label-filter="smoke"` |
| `--dsp-tag` | Data Science Pipelines image tag | `"stable"`                                    | `--dsp-tag="master"` |
| `--dspo-owner` | DSPO repository owner                     | `"opendatahub-io" or "red-hat-data-services"` | `--dspo-owner="myorg"` |
| `--dspo-repo` | DSPO repository name                      | `"data-science-pipelines-operator"`           | `--dspo-repo="custom-repo"` |
| `--dspo-branch` | DSPO branch to deploy                     | `"stable"`                                    | `--dspo-branch="main"` |
| `--deploy-dspo` | Whether to deploy DSPO                    | `"true"`                                      | `--deploy-dspo="false"` |
| `--num-parallel-tests` | Number of parallel test nodes             | `10`                                          | `--num-parallel-tests="5"` |

## Container Usage

### Basic Podman Command Structure

```bash
podman run -it \
  -v $LOCAL_PATH_TO_JUNIT_RESULTS:/dspa/backend/test/end2end/reports:Z \
  -v $LOCAL_PATH_TO_KUBE_CONFIG:/dspa/backend/test/.kube/config:Z \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  [additional environment variables] \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --label-filter="smoke" \
  --num-parallel-tests=5
```

### Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|---------------|---------|
| `$LOCAL_PATH_TO_JUNIT_RESULTS` | `/dspa/backend/test/end2end/reports` | JUnit XML test results output |
| `$LOCAL_PATH_TO_KUBE_CONFIG` | `/dspa/backend/test/.kube/config` | OpenShift cluster authentication |

## Example Commands

### 1. Smoke Tests with Embedded Minio
```bash
podman run -it \
  -v /tmp/test-results:/dspa/backend/test/end2end/reports:Z \
  -v ~/.kube/config:/dspa/backend/test/.kube/config:Z \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --label-filter="smoke" \
  --num-parallel-tests=3
```

### 2. Full Test Suite with S3 Storage
```bash
podman run -it \
  -v /tmp/test-results:/dspa/backend/test/end2end/reports:Z \
  -v ~/.kube/config:/dspa/backend/test/.kube/config:Z \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  -e AWS_ACCESS_KEY_ID="your-key-id" \
  -e AWS_SECRET_ACCESS_KEY="your-secret" \
  -e BUCKET="your-bucket" \
  -e REGION="us-east-1" \
  -e SECRET_NAME="dashboard-dspa-secret" \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --label-filter="E2ECritical" \
  --num-parallel-tests=8
```

### 3. Custom DSPO Testing
```bash
podman run -it \
  -v /tmp/test-results:/dspa/backend/test/end2end/reports:Z \
  -v ~/.kube/config:/dspa/backend/test/.kube/config:Z \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --label-filter="integration" \
  --dspo-owner="my-org" \
  --dspo-branch="feature-branch" \
  --dsp-tag="latest" \
  --num-parallel-tests=5
```

## Upgrade verification (RHOAI / ODH product upgrade)

OpenShift product upgrade jobs run the same Ginkgo suites with `--label-filter=UpgradePreparation` before the
cluster upgrade and `--label-filter=UpgradeVerification` after upgrade (see `backend/test/v2/api/upgrade_api_test.go`
and `upgrade_aipipelines_verification_test.go`). GitHub Actions runs the same labels in
[`.github/workflows/upgrade-test.yml`](../../../.github/workflows/upgrade-test.yml).

`UpgradeVerification` reuses the DSPA namespace and API data created during `UpgradePreparation` (`test-run.sh`).

### Modular AIPipelines acceptance criteria (post-upgrade)

After a platform upgrade, `UpgradeVerification` also checks the cluster-scoped modular **AIPipelines** operand when
the CRD `aipipelines.components.platform.opendatahub.io` is present (skipped on clusters without the modular
module, for example plain Kind KFP upgrade CI):

1. CRD exists and is Established
2. Cluster-scoped `default-aipipelines` exists and is reconciled (`status.observedGeneration == metadata.generation`)
3. `status.phase == Ready`
4. Conditions `Ready`, `ProvisioningSucceeded`, `DSPOReady`, and `ArgoWorkflowsControllersReady` are True with
   matching `observedGeneration`
5. `status.releases[name=platform].version` matches the target build when configured

These checks mirror DSPO modular lifecycle assertions (`tests/aipipelines/fixture_test.go` `waitModule()` in
[data-science-pipelines-operator PR #1112](https://github.com/opendatahub-io/data-science-pipelines-operator/pull/1112)).
Full DSPA lifecycle and module deletion scenarios remain in DSPO integration and `aipipelines-e2e-test`.

Set the expected platform release for Jenkins / `rhoai-test-flow` upgrade stages:

```bash
export EXPECTED_PLATFORM_RELEASE_VERSION="<target platform version>"
```

**Out of scope:** RHOAI downgrade paths are not exercised here; downgrade remains a manual / release-process concern.

## Test Labels Available

Based on the default configuration:

- `UpgradePreparation` - Seed DSPA API data before a product upgrade
- `UpgradeVerification` - Verify DSPA API persistence and modular AIPipelines status after upgrade
- `E2ECritical` - Critical end-to-end tests (default)
- `smoke` - Basic functionality validation
- `integration` - Integration test scenarios
- `regression` - Regression test suite

## Test Execution Flow

The test script performs these steps:

1. **Environment Setup**
   - Sources environment variables from `.env`
   - Generates random namespace suffix (e.g., `dspa-test-a1b2c3d4`)
   - Configures storage type (S3 or Minio)

2. **Resource Deployment**
   - Creates test namespace
   - Deploys Data Science Pipelines Operator (if enabled)
   - Creates DSPA custom resource with name "dspa"
   - Sets up storage secrets (for S3)
   - Creates RBAC resources

3. **Test Execution**
   - Waits for deployment readiness
   - Generates API token for authentication
   - Runs Ginkgo tests with specified filters and parallelism

4. **Cleanup**
   - Removes test namespace and all resources

## Output and Results

### Test Reports
JUnit XML reports are written to the mounted results directory:
```bash
$LOCAL_PATH_TO_JUNIT_RESULTS/
├── junit_01.xml
├── junit_02.xml
└── ...
```

### Exit Codes
- `0`: All tests passed successfully
- `1`: Test failures or deployment errors occurred

## Troubleshooting

### Common Issues

1. **Missing Environment Variables**
   ```bash
   # Error: DSPA_NAME not set
   # Solution: Add -e DSPA_NAME="dspa"
   ```

2. **Storage Configuration**
   ```bash
   # For S3: Ensure all AWS_* variables are set
   # For Minio: Ensure AWS variables are NOT set
   ```

3. **Namespace Conflicts**
   - Random suffix prevents most conflicts
   - Check for existing resources: `oc get dspa -A`

4. **Permission Issues**
   - Ensure cluster-admin or equivalent permissions
   - Check kubeconfig validity: `oc whoami`

### Debug Mode

Add verbose logging:
```bash
podman run -it \
  -e GINKGO_DEBUG=true \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  [other environment variables] \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --num-parallel-tests=1 \
  --label-filter="smoke"
```

## AI Agent Implementation Guide

### Complete Example Workflow

```bash
#!/bin/bash
set -e

# 1. Prepare test environment
export TEST_TIMESTAMP=$(date +%s)
export TEST_RESULTS_DIR="/tmp/dspa-tests-${TEST_TIMESTAMP}"
mkdir -p "$TEST_RESULTS_DIR"

# 2. Set required environment variables
export DSPA_NAME="dspa"
export NAMESPACE="dspa-test"

# 3. Optional: Configure S3 storage (comment out for Minio)
# export AWS_ACCESS_KEY_ID="your-access-key"
# export AWS_SECRET_ACCESS_KEY="your-secret-key"
# export BUCKET="your-test-bucket"
# export REGION="us-east-1"
# export SECRET_NAME="dashboard-dspa-secret"

# 4. Run smoke tests
echo "Running Data Science Pipelines smoke tests..."
podman run -it \
  -v "$TEST_RESULTS_DIR":/dspa/backend/test/end2end/reports:Z \
  -v ~/.kube/config:/dspa/backend/test/.kube/config:Z \
  -e DSPA_NAME="dspa" \
  -e NAMESPACE="dspa-test" \
  ${AWS_ACCESS_KEY_ID:+-e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID"} \
  ${AWS_SECRET_ACCESS_KEY:+-e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY"} \
  ${BUCKET:+-e BUCKET="$BUCKET"} \
  ${REGION:+-e REGION="$REGION"} \
  ${SECRET_NAME:+-e SECRET_NAME="$SECRET_NAME"} \
  --rm \
  --workdir=/dspa/backend/test/images \
  quay.io/opendatahub/ds-pipelines-tests:master \
  ./test-run.sh \
  --label-filter="smoke" \
  --num-parallel-tests=5

# 5. Check results
echo "Test execution completed. Results in: $TEST_RESULTS_DIR"
if ls "$TEST_RESULTS_DIR"/*.xml 1> /dev/null 2>&1; then
    echo "JUnit XML reports generated successfully"
    # Parse results as needed
else
    echo "Warning: No XML reports found"
fi
```

### Key Points for AI Agents

1. **Always set `DSPA_NAME="dspa"` and `NAMESPACE="dspa-test"`** - these are required
2. **Use the exact working directory** - `/dspa/backend/test/images`
3. **Choose storage carefully** - S3 requires credentials, Minio is embedded
4. **Monitor test duration** - start with `--num-parallel-tests=3` and adjust
5. **Check mounted volumes** - ensure paths exist and have correct permissions
6. **Parse JUnit XML** for detailed test results and failure analysis
