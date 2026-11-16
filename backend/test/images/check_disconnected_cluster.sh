#!/bin/bash
# Classify the cluster using OperatorHub disableAllDefaultSources.

check_disconnected_cluster() {
    local disconnected_result
    if ! disconnected_result=$(oc get operatorhub.config.openshift.io cluster \
        -o 'jsonpath={.spec.disableAllDefaultSources}' --request-timeout=15s); then
        echo "ERROR: Cannot read OperatorHub 'cluster'; cannot determine cluster connectivity" >&2
        return 1
    fi

    case "$disconnected_result" in
        true) ;;
        false|"") disconnected_result=false ;;
        *)
            echo "ERROR: Unexpected OperatorHub disableAllDefaultSources value: $disconnected_result" >&2
            return 1
            ;;
    esac

    echo "Disconnected cluster detection: $disconnected_result" >&2
    echo "$disconnected_result"
    return 0
}
