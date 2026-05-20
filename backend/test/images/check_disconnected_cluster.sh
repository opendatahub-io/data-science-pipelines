#!/bin/bash
# Check whether the current Kubernetes/OpenShift cluster is disconnected (air-gapped).
NETWORK_TEST_IMAGE="${NETWORK_TEST_IMAGE:-busybox}"
NETWORK_TEST_TIMEOUT="${NETWORK_TEST_TIMEOUT:-5}"
NETWORK_TEST_URL="${NETWORK_TEST_URL:-http://google.com}"

check_mirror_policies() {
    local idms_output idms_count

    if ! idms_output=$(oc get imagedigestmirrorset -A --no-headers 2>&1); then
        echo "ERROR: 'oc get imagedigestmirrorset' failed: ${idms_output}" >&2
        return 2
    fi

    idms_count=$(echo "$idms_output" | grep -c .)
    if [[ "$idms_count" -gt 0 ]]; then
        echo "Found ${idms_count} ImageDigestMirrorSet resource(s)" >&2
        return 0
    fi

    echo "No image mirror policies found" >&2
    return 1
}

check_network_access() {
    echo "Testing outbound network access from a pod (timeout: ${NETWORK_TEST_TIMEOUT}s)..." >&2
    local output pod_name
    pod_name="disconnected-net-test-$(head /dev/urandom | tr -dc a-z0-9 | head -c 6)"

    output=$(oc run "${pod_name}" \
        --image="${NETWORK_TEST_IMAGE}" \
        --restart=Never \
        --rm \
        --attach \
        --command \
        --request-timeout="${NETWORK_TEST_TIMEOUT}s" \
        -- wget -qO- --timeout="${NETWORK_TEST_TIMEOUT}" "${NETWORK_TEST_URL}" 2>&1) && {
        echo "Pod reached ${NETWORK_TEST_URL} — network is reachable" >&2
        return 1
    }

    if echo "${output}" | grep -qiE 'AlreadyExists|ImagePullBackOff|ErrImagePull|CreateContainerConfigError|Unschedulable'; then
        echo "ERROR: Pod infrastructure failure (not a network test result): ${output}" >&2
        oc delete pod "${pod_name}" --ignore-not-found=true &>/dev/null
        return 2
    fi

    echo "Pod failed to reach ${NETWORK_TEST_URL} — network is unreachable" >&2
    echo "Detail: ${output}" >&2
    oc delete pod "${pod_name}" --ignore-not-found=true &>/dev/null
    return 0
}

check_disconnected_cluster() {
    local context
    context=$(oc config current-context 2>/dev/null) || {
        echo "ERROR: No active kubeconfig context found" >&2
        return 1
    }
    echo "Checking cluster context: ${context}" >&2

    local has_mirrors=false no_network=false mirror_rc network_rc

    check_mirror_policies
    mirror_rc=$?
    if [[ "$mirror_rc" -eq 0 ]]; then
        has_mirrors=true
    elif [[ "$mirror_rc" -eq 2 ]]; then
        echo "ERROR: Failed to query mirror policies, cannot determine cluster state" >&2
        return 1
    fi

    check_network_access
    network_rc=$?
    if [[ "$network_rc" -eq 0 ]]; then
        no_network=true
    elif [[ "$network_rc" -eq 2 ]]; then
        echo "ERROR: Pod infrastructure failure during network check, cannot determine cluster state" >&2
        return 1
    fi

    echo "Result: mirror_policies=${has_mirrors}, network_unreachable=${no_network}" >&2

    if [[ "$has_mirrors" == "true" && "$no_network" == "true" ]]; then
        echo "DISCONNECTED: Mirror policies present AND no outbound network access" >&2
        echo "true"
    else
        echo "CONNECTED: Cluster does not meet both disconnected criteria" >&2
        echo "false"
    fi
    return 0
}
