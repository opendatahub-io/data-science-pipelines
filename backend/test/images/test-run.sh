#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)" || exit 1
cd "$SCRIPT_DIR" || exit 1
REQUESTED_TEST_DIRECTORY="${TEST_DIRECTORY-}"

# Only the dedicated upgrade orchestration labels switch suites; match exact
# labels only (compound API filters use substrings like !UpgradePreparation).
upgrade_test_label() {
  [[ "$1" == "UpgradePreparation" || "$1" == "UpgradeVerification" ]]
}

NAMESPACE_OWNED=false
PREPARATION_COMPLETE=false
FAILURE_MESSAGE=""
dspa_deployment=""
dspa_role_binding=""
API_TOKEN=""
TEST_PROCESS_ID=""
CLEANUP_PROCESS_ID=""
PROCESS_LAUNCHING=false
SHUTDOWN_STATUS=0
# Inspect all arguments before setup so early failures use the requested phase,
# even when an invalid option precedes the label. Validation happens below.
TEST_LABEL="smoke"
for argument in "$@"; do
  case "$argument" in
    --label-filter=*) TEST_LABEL="${argument#*=}" ;;
  esac
done
TEST_PHASE="Tests"
if upgrade_test_label "$TEST_LABEL"; then
  TEST_PHASE="$TEST_LABEL"
fi

# Establish reporting before sourcing setup files or validating arguments.
REPORT_OUTPUT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/end2end/reports"
if ! mkdir -p "$REPORT_OUTPUT_DIR"; then
  echo "ERROR: Cannot create reports directory $REPORT_OUTPUT_DIR" >&2
  exit 1
fi

fail() {
  FAILURE_MESSAGE="$1"
  echo "ERROR: $FAILURE_MESSAGE" >&2
  exit "${2:-1}"
}

redact_diagnostics() {
  local line secret
  while IFS= read -r line || [[ -n "$line" ]]; do
    for secret in "$AWS_ACCESS_KEY_ID" "$AWS_SECRET_ACCESS_KEY" "$AWS_SESSION_TOKEN" "$API_TOKEN"; do
      if [[ -n "$secret" ]]; then
        line=${line//"$secret"/[REDACTED]}
      fi
    done
    printf '%s\n' "$line"
  done
}

collect_diagnostics() {
  echo "Saving diagnostics to $REPORT_OUTPUT_DIR"
  if [[ "$NAMESPACE_OWNED" != "true" ]]; then
    return
  fi
  {
    echo "----- DSPA STATUS -----"
    timeout --foreground 15s oc -n "$NAMESPACE" get dspa "$DSPA_NAME" \
      -o 'jsonpath={.metadata.name}{"\n"}{.metadata.generation}{"\n"}{.status}{"\n"}' --request-timeout=10s \
      || echo "WARNING: Could not collect DSPA status"
    # Reuse the same namespace events, endpoints, and pod log collector as CI.
    # Its console output is saved here so credentials are redacted before writing.
    timeout --foreground 2m bash "$SCRIPT_DIR/../../../.github/resources/scripts/collect-logs.sh" \
      --ns "$NAMESPACE" --output /dev/null \
      || echo "WARNING: Pod log collection failed or timed out; continuing cleanup"
  } 2>&1 | redact_diagnostics | tee "$REPORT_OUTPUT_DIR/${TEST_PHASE}-pod-logs.txt"
}

write_harness_report() {
  local result="$1"
  printf 'Phase: %s\nNamespace: %s\nExit status: %s\n%s\n' \
    "$TEST_PHASE" "$NAMESPACE" "$result" "$FAILURE_MESSAGE" \
    | redact_diagnostics > "$REPORT_OUTPUT_DIR/${TEST_PHASE}-result.txt"

  printf 'Final exit status: %s\n' "$result" >> "$REPORT_OUTPUT_DIR/${TEST_PHASE}-result.txt"
  # Ginkgo cannot report setup or compilation failures. Always publish the
  # harness result so a passing retry replaces any failure stashed by Jenkins.
  local harness_failures=0
  [[ "$result" -eq 0 ]] || harness_failures=1
  {
    printf '<?xml version="1.0" encoding="UTF-8"?>\n'
    printf '<testsuite name="DSPA %s harness" tests="1" failures="%s" errors="0" skipped="0">\n' "$TEST_PHASE" "$harness_failures"
    printf '  <testcase classname="DSPA.%s.Harness" name="Complete test runner lifecycle">\n' "$TEST_PHASE"
    if [[ "$result" -ne 0 ]]; then
      printf '    <failure type="HarnessFailure" message="Test runner exited with status %s">' "$result"
      redact_diagnostics < "$REPORT_OUTPUT_DIR/${TEST_PHASE}-result.txt" \
        | LC_ALL=C tr -d '\000-\010\013\014\016-\037' \
        | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'
      printf '    </failure>\n'
    fi
    printf '  </testcase>\n</testsuite>\n'
  } > "$REPORT_OUTPUT_DIR/${TEST_PHASE}-harness-junit.xml" || {
    echo "ERROR: Could not write harness xUnit report" >&2
    return 1
  }
}

stop_process_group() {
  local process_id="$1" grace_seconds="${2:-5}"
  [[ -n "$process_id" ]] || return 0
  # Also signal the direct child in case termination races with group startup.
  kill -TERM -- "-$process_id" 2>/dev/null || true
  kill -TERM "$process_id" 2>/dev/null || true
  local shutdown_deadline=$((SECONDS + grace_seconds))
  while kill -0 -- "-$process_id" 2>/dev/null || kill -0 "$process_id" 2>/dev/null; do
    if (( SECONDS >= shutdown_deadline )); then
      kill -KILL -- "-$process_id" 2>/dev/null || true
      kill -KILL "$process_id" 2>/dev/null || true
      break
    fi
    sleep 0.1
  done
  wait "$process_id" 2>/dev/null || true
}

run_cleanup_command() {
  local timeout_seconds="$1" command_status initial_shutdown_status="$SHUTDOWN_STATUS"
  shift
  local command_deadline=$((SECONDS + timeout_seconds))
  PROCESS_LAUNCHING=true
  # Job control gives this background subshell its own group, including shell
  # functions and pipelines. Disable it inside so their children keep that group.
  # timeout --foreground likewise keeps diagnostic/deletion commands in it.
  set -m
  (set +m; "$@") &
  CLEANUP_PROCESS_ID=$!
  set +m
  PROCESS_LAUNCHING=false
  if [[ "$initial_shutdown_status" -eq 0 && "$SHUTDOWN_STATUS" -ne 0 ]]; then
    finish "$SHUTDOWN_STATUS"
  fi
  # Bound the whole group as well as individual timeout commands: --foreground
  # cannot itself terminate grandchildren that outlive a diagnostic shell.
  # Short sleeps keep signal traps responsive while enforcing that deadline.
  while kill -0 "$CLEANUP_PROCESS_ID" 2>/dev/null; do
    if (( SECONDS >= command_deadline )); then
      stop_process_group "$CLEANUP_PROCESS_ID" 0
      CLEANUP_PROCESS_ID=""
      return 124
    fi
    sleep 0.1
  done
  wait "$CLEANUP_PROCESS_ID"
  command_status=$?
  CLEANUP_PROCESS_ID=""
  return "$command_status"
}

terminate() {
  [[ "$SHUTDOWN_STATUS" -eq 0 ]] || return 0
  SHUTDOWN_STATUS="$1"
  FAILURE_MESSAGE="$2"
  # A signal between starting the child and saving $! must not orphan it.
  [[ "$PROCESS_LAUNCHING" == "false" ]] || return 0
  # Enter shutdown directly, including from an already-running EXIT handler.
  # finish exits, so the suspended normal cleanup path can never resume.
  finish "$SHUTDOWN_STATUS"
}

finish() {
  local result="${1:-$?}"
  trap - EXIT
  # Only ignore repeat signals after an interruption has already selected the
  # bounded shutdown path. Normal cleanup must remain interruptible.
  if [[ "$SHUTDOWN_STATUS" -ne 0 ]]; then
    trap '' INT TERM
  fi
  if [[ "$result" -ne 0 && -z "$FAILURE_MESSAGE" ]]; then
    FAILURE_MESSAGE="Test runner exited with status $result"
  fi
  if [[ "$SHUTDOWN_STATUS" -ne 0 ]]; then
    # Publish before waiting for workers or making any cluster requests.
    write_harness_report "$result" || echo "ERROR: Could not publish interruption report" >&2
  fi
  stop_process_group "$CLEANUP_PROCESS_ID"
  CLEANUP_PROCESS_ID=""
  stop_process_group "$TEST_PROCESS_ID"
  TEST_PROCESS_ID=""
  if [[ "$result" -eq 0 && "$PREPARATION_COMPLETE" == "true" ]]; then
    echo "Retaining successful preparation namespace $NAMESPACE for upgrade verification"
  else
    if [[ "$SHUTDOWN_STATUS" -eq 0 ]]; then
      run_cleanup_command 135 collect_diagnostics || echo "WARNING: Diagnostic collection failed; continuing cleanup" >&2
    fi
    if [[ "$NAMESPACE_OWNED" == "true" ]]; then
      echo "Deleting namespace: $NAMESPACE"
      local cleanup_status=0
      if [[ "$SHUTDOWN_STATUS" -ne 0 ]]; then
        # Submit deletion without waiting for finalizers during interruption.
        run_cleanup_command 4 timeout --foreground --kill-after=1s 3s oc delete namespace "$NAMESPACE" --ignore-not-found=true \
          --wait=false --request-timeout=2s || cleanup_status=$?
      else
        run_cleanup_command 120 timeout --foreground 2m oc delete namespace "$NAMESPACE" --ignore-not-found=true \
          --timeout=2m --request-timeout=30s || cleanup_status=$?
      fi
      if [[ "$cleanup_status" -ne 0 ]]; then
        echo "ERROR: Namespace cleanup failed for $NAMESPACE; remove it before retrying" >&2
        FAILURE_MESSAGE="${FAILURE_MESSAGE}${FAILURE_MESSAGE:+; }Namespace cleanup failed; manual cleanup required."
        [[ "$result" -ne 0 ]] || result=1
      fi
    fi
  fi
  [[ -z "$dspa_deployment" ]] || rm -f "$dspa_deployment"
  [[ -z "$dspa_role_binding" ]] || rm -f "$dspa_role_binding"
  write_harness_report "$result" || { [[ "$result" -ne 0 ]] || result=1; }
  exit "$result"
}
trap finish EXIT
trap 'terminate 130 "Test runner interrupted"' INT
trap 'terminate 143 "Test runner terminated"' TERM

set -o allexport
source .env || fail "Cannot load test runner configuration .env"
source "./patch-csv.sh" || fail "Cannot load CSV patch helper"
set +o allexport

if [ -n "$REQUESTED_TEST_DIRECTORY" ]; then
  TEST_DIRECTORY="$REQUESTED_TEST_DIRECTORY"
fi

export DEPLOYMENT_NAME="ds-pipeline-$DSPA_NAME"

SKIP_DEPLOYMENT="${SKIP_DEPLOYMENT:-true}"

TEST_LABEL="smoke"
REQUESTED_TEST_LABEL=""
DSPO_OWNER="opendatahub-io"
DSPO_REPO="data-science-pipelines-operator"
DSPO_BRANCH="stable"
DSP_TAG="stable"
DEPLOY_DSPO="true"
NUM_PARALLEL_NODES=10
CUSTOM_PIP_INDEX_URL="https://pypi.org/simple"
CUSTOM_PIP_TRUSTED_HOST="pypi.org"
GPU_VENDOR="nvidia"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --label-filter=*)
      TEST_LABEL="${1#*=}"
      REQUESTED_TEST_LABEL="$TEST_LABEL"
      shift
      ;;
    --dsp-tag=*)
      DSP_TAG="${1#*=}"
      shift
      ;;
    --dspo-owner=*)
      DSPO_OWNER="${1#*=}"
      shift
      ;;
    --dspo-repo=*)
      DSPO_REPO="${1#*=}"
      shift
      ;;
    --dspo-branch=*)
      DSPO_BRANCH="${1#*=}"
      shift
      ;;
    --deploy-dspo=*)
      DEPLOY_DSPO="${1#*=}"
      shift
      ;;
    --num-parallel-tests=*)
      NUM_PARALLEL_NODES="${1#*=}"
      shift
      ;;
    --gpu-vendor=*)
      GPU_VENDOR="${1#*=}"
      shift
      ;;
    *)
      fail "Unknown argument $1"
      ;;
  esac
done

DSPA_READY_TIMEOUT="${DSPA_READY_TIMEOUT:-10m}"
BASE_NAMESPACE="${NAMESPACE-}"
if upgrade_test_label "$TEST_LABEL"; then
  TEST_DIRECTORY="v2/api"
else
  RANDOM_SUFFIX="${RANDOM}-${RANDOM}"
  export NAMESPACE="$NAMESPACE-$RANDOM_SUFFIX"
fi

[[ -n "$BASE_NAMESPACE" && -n "$DSPA_NAME" ]] || fail "NAMESPACE and DSPA_NAME must be set"
[[ "$DSPA_READY_TIMEOUT" =~ ^[1-9][0-9]*[smh]$ ]] || \
  fail "DSPA_READY_TIMEOUT must be a positive integer followed by s, m, or h (for example 10m)"

# Resolve the suite before creating any resources so a bad TEST_DIRECTORY cannot
# leave an otherwise successful preparation namespace behind.
current_dir="$SCRIPT_DIR"
found_test_dir=""
while [[ "$current_dir" != "/" ]]; do
  if [[ -d "$current_dir/$TEST_DIRECTORY" ]]; then
    found_test_dir="$current_dir/$TEST_DIRECTORY"
    break
  fi
  current_dir="$(dirname "$current_dir")"
done
[[ -n "$found_test_dir" ]] || fail "Could not find test directory $TEST_DIRECTORY"

if [[ "$TEST_LABEL" == "UpgradeVerification" ]]; then
  echo "UpgradeVerification: checking preparation state in namespace $NAMESPACE"
  preparation_state=$(oc get namespace "$NAMESPACE" --request-timeout=30s \
    -o 'jsonpath={.metadata.labels.pipelines\.kubeflow\.org/test-runner}{"|"}{.metadata.annotations.pipelines\.kubeflow\.org/upgrade-preparation}') \
    || fail "UpgradeVerification requires the successful preparation namespace $NAMESPACE"
  [[ "${preparation_state%%|*}" == "true" ]] || \
    fail "Namespace $NAMESPACE is not marked as owned by the test runner; refusing to reuse or delete it"
  [[ "$preparation_state" == "true|complete" ]] || \
    fail "UpgradePreparation did not complete successfully in namespace $NAMESPACE"
  oc -n "$NAMESPACE" get dspa "$DSPA_NAME" --request-timeout=30s >/dev/null \
    || fail "UpgradeVerification requires preparation DSPA $NAMESPACE/$DSPA_NAME"
  NAMESPACE_OWNED=true
fi

source "./check_disconnected_cluster.sh" || fail "Cannot load disconnected detection helper"
if ! DISCONNECTED_RESULT="$(check_disconnected_cluster 2>"$REPORT_OUTPUT_DIR/${TEST_PHASE}-detection.log")"; then
  cat "$REPORT_OUTPUT_DIR/${TEST_PHASE}-detection.log" >&2
  fail "Automatic disconnected detection failed; no storage configuration selected"
fi
cat "$REPORT_OUTPUT_DIR/${TEST_PHASE}-detection.log" >&2
export DISCONNECTED_CLUSTER="$DISCONNECTED_RESULT"
if [[ "$DISCONNECTED_RESULT" == "true" ]]; then
  echo "Disconnected cluster detected"
  if ! upgrade_test_label "$TEST_LABEL"; then
    if [ -z "$REQUESTED_TEST_LABEL" ]; then
      TEST_LABEL="disconnected"
    elif [[ "$REQUESTED_TEST_LABEL" == "smoke" ]]; then
      if [[ "$TEST_DIRECTORY" == "v2/api" ]]; then
        TEST_LABEL="smoke"
      else
        TEST_LABEL="smoke && disconnected"
      fi
    fi
    # Any other explicit --label-filter is unchanged. --fail-on-empty fails the
    # lane if it matches zero specs on the air-gap cluster.
  fi
  CLUSTER_BASE_DOMAIN=""
  for attempt in 1 2 3; do
    CLUSTER_BASE_DOMAIN=$(oc get dns cluster -o jsonpath='{.spec.baseDomain}' --request-timeout=30s 2>/dev/null) && break
    echo "WARNING: oc get dns cluster failed (attempt $attempt/3), retrying in 5s..."
    sleep 5
  done
  if [[ -z "$CLUSTER_BASE_DOMAIN" ]]; then
    fail "Failed to retrieve CLUSTER_BASE_DOMAIN from 'oc get dns cluster' after 3 attempts"
  fi
  BASTION_HOST="bastion.${CLUSTER_BASE_DOMAIN}"
  AWS_ENDPOINT="${BASTION_HOST}:9000"
  AWS_SECRET_ACCESS_KEY="minioadmin"
  AWS_ACCESS_KEY_ID="minioadmin"
  BUCKET="ods-ci-ds-pipelines"
  REGION="us-east-1"
  CUSTOM_PIP_INDEX_URL="https://${BASTION_HOST}:9443/root/pypi/+simple/"
  CUSTOM_PIP_TRUSTED_HOST="${BASTION_HOST}"
elif ! upgrade_test_label "$TEST_LABEL"; then
  if [[ "$TEST_LABEL" == "smoke" ]]; then
    TEST_LABEL='smoke && !disconnected'
  fi
fi

dspa_deployment=$(mktemp) || fail "Cannot create temporary DSPA manifest"
dspa_role_binding=$(mktemp) || fail "Cannot create temporary role binding manifest"

# Check if AWS credentials are provided to determine storage type
if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" ]]; then
  STORAGE_TYPE="s3"
  echo "Using external S3 storage (AWS credentials detected)"
else
  STORAGE_TYPE="minio"
  echo "Using embedded Minio storage (no AWS credentials provided)"
fi

# Create DSPA CR template
cat <<EOF >> "$dspa_deployment"
apiVersion: datasciencepipelinesapplications.opendatahub.io/v1
kind: DataSciencePipelinesApplication
metadata:
  name: $DSPA_NAME
  namespace: $NAMESPACE
spec:
  dspVersion: v2
EOF

if [ "$SKIP_DEPLOYMENT" = "false" ]; then
  cat <<EOF >> "$dspa_deployment"
  apiServer:
    image: "quay.io/opendatahub/ds-pipelines-api-server:$DSP_TAG"
    argoDriverImage: "quay.io/opendatahub/ds-pipelines-driver:$DSP_TAG"
    argoLauncherImage: "quay.io/opendatahub/ds-pipelines-launcher:$DSP_TAG"
    cacheEnabled: true
    enableSamplePipeline: false
  persistenceAgent:
    image: "quay.io/opendatahub/ds-pipelines-persistenceagent:$DSP_TAG"
  scheduledWorkflow:
    image: "quay.io/opendatahub/ds-pipelines-scheduledworkflow:$DSP_TAG"
EOF
else
  cat <<EOF >> "$dspa_deployment"
  apiServer:
    cacheEnabled: true
    enableSamplePipeline: false
EOF
fi

# Add storage-specific configuration
if [ "$STORAGE_TYPE" = "s3" ]; then
  cat <<EOF >> "$dspa_deployment"
  objectStorage:
    externalStorage:
      bucket: $BUCKET
      host: $AWS_ENDPOINT
      region: $REGION
      s3CredentialsSecret:
        accessKey: AWS_ACCESS_KEY
        secretKey: AWS_SECRET_ACCESS_KEY
        secretName: $SECRET_NAME
      scheme: https
  podToPodTLS: true
EOF
else
  cat <<EOF >> "$dspa_deployment"
  objectStorage:
    minio:
      deploy: true
      image: 'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
EOF
fi

cat <<EOF >> "$dspa_role_binding"
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: dashboard-permissions-$DSPA_NAME
  namespace: $NAMESPACE
subjects:
  - kind: ServiceAccount
    name: $DEPLOYMENT_NAME
    namespace: $NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
EOF

#################### PATCH CSV #######################
if [ "$DEPLOY_DSPO" == "true" ] && [ "$SKIP_DEPLOYMENT" = "false" ]; then
  find_csv_and_update "$DSPO_OWNER" "$DSPO_REPO" "$DSPO_BRANCH" || fail "Failed to prepare operator manifests"
fi

#################### DEPLOY DSPA #######################

if [[ "$TEST_LABEL" != "UpgradeVerification" ]]; then
  echo "Creating namespace: $NAMESPACE"
  oc create namespace "$NAMESPACE" --request-timeout=30s \
    || fail "Failed to create namespace $NAMESPACE; existing namespaces will not be deleted"
  NAMESPACE_OWNED=true
  oc label namespace "$NAMESPACE" pipelines.kubeflow.org/test-runner=true --request-timeout=30s \
    || fail "Failed to mark test namespace $NAMESPACE"

  if [[ "$STORAGE_TYPE" == "s3" ]]; then
    echo "Creating AWS credentials secret for S3 storage"
    oc -n "$NAMESPACE" create secret generic "$SECRET_NAME" \
      --from-literal=AWS_ACCESS_KEY="$AWS_ACCESS_KEY_ID" \
      --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" --request-timeout=30s \
      || fail "Failed to create object storage credentials Secret in $NAMESPACE"
  fi

  echo "Creating DSPA $NAMESPACE/$DSPA_NAME"
  oc apply -n "$NAMESPACE" -f "$dspa_deployment" --request-timeout=30s \
    || fail "Failed to apply DSPA $NAMESPACE/$DSPA_NAME"
fi

# DSPA status covers database/object-storage prerequisites and the workloads;
# deployment creation can legitimately take more than a minute during cold start.
echo "Waiting up to $DSPA_READY_TIMEOUT for DSPA $NAMESPACE/$DSPA_NAME to become ready"
timeout "$DSPA_READY_TIMEOUT" oc -n "$NAMESPACE" wait --for=condition=Ready "dspa/$DSPA_NAME" \
  --timeout="$DSPA_READY_TIMEOUT" --request-timeout="$DSPA_READY_TIMEOUT" \
  || fail "DSPA $NAMESPACE/$DSPA_NAME did not become ready within $DSPA_READY_TIMEOUT"
echo "DSPA $NAMESPACE/$DSPA_NAME is ready"

if [[ "$TEST_LABEL" != "UpgradeVerification" ]]; then
  echo "Creating role binding for DSPA API access"
  oc apply -f "$dspa_role_binding" -n "$NAMESPACE" --request-timeout=30s \
    || fail "Failed to create DSPA API role binding in $NAMESPACE"
fi

API_HOST=$(oc -n "$NAMESPACE" get route "$DEPLOYMENT_NAME" -o jsonpath='{.spec.host}' --request-timeout=30s) \
  || fail "Failed to fetch route $NAMESPACE/$DEPLOYMENT_NAME"
[[ -n "$API_HOST" ]] || fail "Route $NAMESPACE/$DEPLOYMENT_NAME has no hostname"
export API_URL="https://$API_HOST"

echo "Generating API token for $DEPLOYMENT_NAME"
API_TOKEN=$(oc create token "$DEPLOYMENT_NAME" --namespace "$NAMESPACE" --duration=60m --request-timeout=30s) \
  || fail "Failed to create API token for $NAMESPACE/$DEPLOYMENT_NAME"
[[ -n "$API_TOKEN" ]] || fail "API token for $NAMESPACE/$DEPLOYMENT_NAME is empty"
export API_TOKEN

#################### TESTS #######################

echo "Found test directory at: $found_test_dir"
echo "Reports directory: $REPORT_OUTPUT_DIR"
cd "$found_test_dir" || fail "Cannot enter test directory $found_test_dir"
echo "Running Tests now..."
command -v setsid >/dev/null || fail "Test runner requires setsid (util-linux)"
# Keep job control disabled so the asynchronous child is not already a group
# leader: setsid then execs Go without forking, and $! is also the group ID.
set +m
PROCESS_LAUNCHING=true
setsid go run github.com/onsi/ginkgo/v2/ginkgo -r -v -p \
  --nodes=$NUM_PARALLEL_NODES \
  --keep-going \
  --fail-on-empty \
  --label-filter="$TEST_LABEL" \
  -- -namespace="$NAMESPACE" \
  -apiUrl="$API_URL" \
  -authToken="$API_TOKEN" \
  -disableTlsCheck=true \
  -gpuVendor=$GPU_VENDOR \
  -customPipIndexURL=$CUSTOM_PIP_INDEX_URL \
  -customPipTrustedHost=$CUSTOM_PIP_TRUSTED_HOST \
  -serviceAccountName=pipeline-runner-"$DSPA_NAME" \
  -baseImage="registry.redhat.io/ubi9/python-312@sha256:e80ff3673c95b91f0dafdbe97afb261eab8244d7fd8b47e20ffcbcfee27fb168" \
  -disconnectedCluster="${DISCONNECTED_CLUSTER:-false}" \
  -reportOutputDir="$REPORT_OUTPUT_DIR" \
  ${EXPECTED_PLATFORM_RELEASE_VERSION:+-expectedPlatformReleaseVersion="$EXPECTED_PLATFORM_RELEASE_VERSION"} &
TEST_PROCESS_ID=$!
PROCESS_LAUNCHING=false
[[ "$SHUTDOWN_STATUS" -eq 0 ]] || exit "$SHUTDOWN_STATUS"

wait "$TEST_PROCESS_ID"
test_status=$?
[[ "$test_status" -eq 0 ]] || fail "Ginkgo tests failed with exit status $test_status" "$test_status"

if [[ "$TEST_LABEL" == "UpgradePreparation" ]]; then
  oc annotate namespace "$NAMESPACE" pipelines.kubeflow.org/upgrade-preparation=complete \
    --overwrite --request-timeout=30s || fail "Failed to mark UpgradePreparation complete"
  PREPARATION_COMPLETE=true
fi
