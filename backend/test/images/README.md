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
  --label-filter="integration" \
  --dspo-owner="my-org" \
  --dspo-branch="feature-branch" \
  --dsp-tag="latest" \
  --num-parallel-tests=5
```

## Disconnected cluster detection

`check_disconnected_cluster.sh` reads
`OperatorHub/cluster.spec.disableAllDefaultSources`:

- `true`: use disconnected-cluster settings.
- `false` or an absent field: use connected-cluster settings.
- A missing OperatorHub resource, failed query, or unexpected value: stop setup
  without selecting storage settings. The API request has a 15-second timeout.

This follows the OperatorHub configuration convention used by the disconnected
provisioning workflow. Detection does not create a pod, pull an image, or test
outbound network access. Environment flags and mirror-policy presence do not
override the OperatorHub result.

On disconnected clusters, the runner configures bastion MinIO and the pip mirror.
Detection messages and query errors are saved as `<phase>-detection.log` directly
in the mounted reports directory. CLI warnings stay in this log and do not change
the Boolean result of a successful query.

## DSPA readiness and setup failures

The runner waits for DSPA `Ready=True` before fetching the route and generating an
API token. Set `DSPA_READY_TIMEOUT` to a positive integer followed by `s`, `m`, or
`h` to change the default `10m` readiness budget, for example:

```bash
export DSPA_READY_TIMEOUT=15m
```

Resource creation, readiness, role binding, route lookup, and token generation
must succeed before Ginkgo starts. On failure, the runner saves diagnostics under
`backend/test/end2end/reports/` and deletes its owned
test namespace. Diagnostic collection is bounded and does not replace the
original failure. Cleanup failures are reported and fail an otherwise successful
run.

The runner reuses `.github/resources/scripts/collect-logs.sh`, the collector used
by CI, to capture namespace events, service endpoints, and pod logs. These are
saved with DSPA status as `<phase>-pod-logs.txt` directly in the reports directory.
The exit-status summary is saved as `<phase>-result.txt`. Phase-specific filenames
keep preparation and verification diagnostics separate.
Every run also writes `<phase>-harness-junit.xml` with a harness test case so setup,
compilation, and cleanup failures appear in Jenkins test results even when Ginkgo
cannot produce a report. This supplements Ginkgo's reports. A successful retry
writes a passing case with the same filename and test identity, replacing the
previous failure when Jenkins merges stashed results.
Reporting is initialized before sourcing `.env` or `patch-csv.sh` and validating
arguments. Failures in these steps also generate a harness report for the
requested upgrade phase, regardless of argument order.
Failure text is XML-escaped and stripped of XML-invalid ASCII control characters.
Known object-storage credentials and the API token are redacted before writing.
Successful verification and normal runs intentionally collect diagnostics before
deleting their namespace, preserving final status and logs for investigation.
Pod log collection has a two-minute timeout so cleanup can still proceed if a
diagnostic command stalls.

The suite runs in a separate process group using `setsid` from `util-linux`.
TERM or INT sent only to the runner interrupts its wait and immediately writes
the harness failure report (exit status 143 or 130). The runner sends TERM to the
suite group, waits up to five seconds, then sends KILL to remaining workers and
reaps its child before requesting namespace deletion. Interruption skips pod-log
collection and submits deletion with `--wait=false`, a two-second API request
timeout, and an outer three-second timeout with a one-second kill allowance.
Allow at least 15 seconds of container termination grace for this shutdown path;
the runner does not wait for namespace finalizers. Normal diagnostic collection
and namespace deletion also run as tracked background process groups. TERM or
INT during either step publishes the interruption report, stops that group, and
switches to the same bounded deletion path. Ordinary completion retains the
diagnostics and cleanup behavior described above. Foreground setup commands can
still defer signal handling until they return.
The entire diagnostic process group has a 135-second deadline (15 seconds for
DSPA status plus two minutes for pod logs), which also stops descendants left
behind by a diagnostic command. Repeat signals during interrupted shutdown do
not change the first signal's exit status or restart its grace period.

## Upgrade verification (RHOAI / ODH product upgrade)

OpenShift product upgrade jobs run the same Ginkgo suites with `--label-filter=UpgradePreparation` before the
cluster upgrade and `--label-filter=UpgradeVerification` after upgrade (see `backend/test/v2/api/upgrade_api_test.go`
and `upgrade_aipipelines_verification_test.go`). Those specs are also tagged **Tier1** for the quality-gate sheet.
Standard **Tier1** API or E2E jobs on connected clusters should use
`Tier1 && !UpgradePreparation && !UpgradeVerification` so upgrade orchestration stays on dedicated upgrade lanes only.
**Kind CI E2E Tier1** uses `Tier1 && !integration` (Ray integration specs need a cluster with that CRD; see `.github/workflows/e2e-test.yml` and `AGENTS.md`)—distinct from the API compound filter above.
GitHub Actions runs upgrade labels in
[`.github/workflows/upgrade-test.yml`](../../../.github/workflows/upgrade-test.yml).

`UpgradeVerification` reuses the DSPA namespace and API data created during `UpgradePreparation` (`test-run.sh`).
Successful preparation marks the namespace with the ownership label
`pipelines.kubeflow.org/test-runner=true` and completion annotation
`pipelines.kubeflow.org/upgrade-preparation=complete`, then retains it. Failed
preparation collects diagnostics and cleans up. Verification requires these
markers and the preparation DSPA; it never recreates missing upgrade state.
Use the updated harness for both phases. After verification succeeds or fails,
the runner saves diagnostics and cleans up the owned namespace.

### `test-run.sh` suite directory (`TEST_DIRECTORY`)

- **Default:** `.env` sets `TEST_DIRECTORY=end2end` (product E2E gates such as `smoke`, `Tier1`, or `E2ECritical`).
- **Upgrade only:** `--label-filter=UpgradePreparation` or `--label-filter=UpgradeVerification` switches to `v2/api` and skips the random namespace suffix so verification reuses preparation data.
- **Normal API Tier1:** compound filters such as `Tier1 && !UpgradePreparation && !UpgradeVerification` do **not** change the suite; set `TEST_DIRECTORY=v2/api` in the environment (or `.env`) and run from the test image with that label filter. Substring checks on `Upgrade` inside compound filters are intentionally **not** used.
- **Disconnected (air-gap) cluster:** runtime setup (bastion MinIO, pip mirror, `-disconnectedCluster=true`) applies to all non-upgrade runs. Label selection: no explicit `--label-filter` → `disconnected`; explicit **`smoke`** with default E2E suite → `smoke && disconnected`; explicit **`smoke`** with `TEST_DIRECTORY=v2/api` stays **`smoke`** (API specs carry no `disconnected` label). **Connected** clusters: default or explicit **`smoke`** → `smoke && !disconnected` in `test-run.sh` (matches Kind CI). Any other explicit filter is unchanged; there may be zero matching specs (e.g. Tier1 on air-gap today), and **`--fail-on-empty`** fails the lane in that case.

### Modular AIPipelines acceptance criteria (post-upgrade)

After a platform upgrade, `UpgradeVerification` checks the cluster-scoped modular **AIPipelines** operand when
the CRD `aipipelines.components.platform.opendatahub.io` is present. If the CRD is absent and
`EXPECTED_PLATFORM_RELEASE_VERSION` is set (required on RHOAI/ODH product upgrade jobs), the test **fails**. If
both the CRD and that variable are unset, the spec is **skipped** (for example Kind KFP upgrade CI without the
modular module):

1. CRD exists and is Established
2. Cluster-scoped `default-aipipelines` exists and is reconciled (`status.observedGeneration == metadata.generation`)
3. `status.phase == Ready`
4. Conditions `Ready`, `ProvisioningSucceeded`, `DSPOReady`, and `ArgoWorkflowsControllersReady` are True with
   matching `observedGeneration`
5. `status.releases[name=platform].version` matches `EXPECTED_PLATFORM_RELEASE_VERSION`

Full DSPA lifecycle and module deletion scenarios remain in DSPO integration and `aipipelines-e2e-test`.

**Required on Jenkins / `rhoai-test-flow` `UpgradeVerification`:** set the target platform release so missing
CRD or a stale handshake fails closed:

```bash
export EXPECTED_PLATFORM_RELEASE_VERSION="<target platform version>"
```

**GitHub Actions Kind CI:** the target `deploy` action exports the `platformVersion` from DSPO's modular
AIPipelines fixture, and `UpgradeVerification` receives it through `expected_platform_release_version`.

**Out of scope:** RHOAI downgrade paths are not exercised here; downgrade remains a manual / release-process concern.

## Test Labels Available

The legacy Ginkgo label **`Sanity`** and **`--label-filter=sanity`** are removed (Phase 2 quality gates). Migrate jobs as follows: **`smoke`** for the Smoke gate; **`Tier1`** with **`./backend/test/end2end`** for product E2E (essential + explicit Tier1 + integration—not a 1:1 Sanity rename); **`Tier1 && !UpgradePreparation && !UpgradeVerification`** with **`./backend/test/v2/api`** for normal API Tier1. Details: [PR 467](https://github.com/opendatahub-io/data-science-pipelines/pull/467) and `AGENTS.md`.

Based on the default configuration:

- `smoke` - Component-level critical tests (default for RHOAI nightly)
- `disconnected` - Air-gapped E2E lane (subset of Smoke IRs in `e2eDisconnectedPipelinePaths`; also tagged `Smoke`; connected smoke uses `smoke && !disconnected`)
- `Tier1` - High-priority tests excluding Smoke
- `Tier2` - Medium- and low-priority positive tests
- `Tier3` - Negative and destructive tests
- `E2ECritical` - Multi-user critical lane; gate IRs also carry `smoke` or `Tier1` on the same spec (remaining `valid/critical` sweep stays `E2ECritical` + `Tier2`)
- `integration` - Cross-component integration (also tagged Tier1; Kind CI runs `Tier1 && !integration`)
- `gpu` - Accelerator-required E2E (also tagged Tier2; CPU/Kind Tier2 uses `Tier2 && !gpu && !dra-check`)
- `dra-check` - DRA scheduling checks (also tagged Tier2; excluded from Kind Tier2 via `!dra-check`)
- `WorkflowCompiler` - Compiler golden tests (also tagged Tier1; GitHub `compiler-tests.yml` keeps this label)
- `WorkflowCompilerVisits` - Compiler visitor iteration (also tagged Tier2)
- `UpgradePreparation` - Seed DSPA API data before a product upgrade (also tagged Tier1; standard Tier1 API jobs use `Tier1 && !UpgradePreparation && !UpgradeVerification`)
- `UpgradeVerification` - Verify DSPA API persistence and modular AIPipelines status after upgrade (also tagged Tier1)

## Test Execution Flow

The test script performs these steps:

1. **Environment Setup**
   - Sources environment variables from `.env`
   - Generates a random namespace suffix except for dedicated upgrade phases
   - Reads OperatorHub configuration to select connected or disconnected settings
   - Configures storage type (S3 or Minio)

2. **Resource Deployment**
   - Creates test namespace
   - Deploys Data Science Pipelines Operator (if enabled)
   - Creates DSPA custom resource with name "dspa"
   - Sets up storage secrets (for S3)
   - Creates RBAC resources

3. **Test Execution**
   - Waits for DSPA `Ready=True` within `DSPA_READY_TIMEOUT`
   - Generates API token for authentication
   - Runs Ginkgo tests with specified filters and parallelism

4. **Cleanup**
   - Retains successful upgrade preparation for the verification phase
   - Otherwise saves diagnostics and removes the owned test namespace

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
