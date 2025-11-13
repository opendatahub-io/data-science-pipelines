#!/bin/bash

set -o allexport
source .env
set +o allexport

# 1. Create a temporary file to store dspa config
temp_file=$(mktemp)

# Create DSPA CR template
cat <<EOF >> "$temp_file"
apiVersion: datasciencepipelinesapplications.opendatahub.io/v1
kind: DataSciencePipelinesApplication
metadata:
  name: $DSPA_NAME
  namespace: $NAMESPACE
spec:
  dspVersion: v2
  apiServer:
    cacheEnabled: true
    enableSamplePipeline: false
  objectStorage:
    externalStorage:
      bucket: $BUCKET
      host: $ENDPOINT
      region: $REGION
      s3CredentialsSecret:
        accessKey: AWS_ACCESS_KEY
        secretKey: AWS_SECRET_ACCESS_KEY
        secretName: $SECRET_NAME
      scheme: https
  podToPodTLS: true
EOF

# Create Namespace
oc create namespace $NAMESPACE

# Create AWS Secret
oc -n $NAMESPACE create secret generic $SECRET_NAME --from-literal=AWS_ACCESS_KEY=$AWS_ACCESS_KEY_ID --from-literal=AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY

# Create DSPA deployment
oc apply -n $NAMESPACE -f $temp_file
timeout=120
SECONDS=0
until oc get deployment/ds-pipeline-$DSPA_NAME --ignore-not-found &> /dev/null || (( SECONDS >= timeout )); do
  echo "Waiting for deployment ds-pipeline-$DSPA_NAME to appear..."
  sleep 10
done
echo "deployment/ds-pipeline-$DSPA_NAME found. Waiting for exit condition..."
oc wait --for=condition=available deployment/ds-pipeline-$DSPA_NAME --timeout=10m

# Get API URL
export API_URL="https://$(oc -n $NAMESPACE get route "ds-pipeline-$DSPA_NAME" -o jsonpath={.spec.host})"

# Get API Token
export API_TOKEN=$(oc create token "ds-pipeline-$DSPA_NAME" --namespace "$NAMESPACE" --duration=60m)

# Run Tests
cd $TEST_DIRECTORY
go run github.com/onsi/ginkgo/v2/ginkgo -r -v --cover -p --keep-going "$@" -- -namespace=$NAMESPACE -apiUrl=$API_URL -authToken="$API_TOKEN" -disableTlsCheck=true -serviceAccountName=pipeline-runner-$DSPA_NAME -repoName="opendatahub-io/data-science-pipelines" -baseImage="registry.redhat.io/ubi9/python-312@sha256:e80ff3673c95b91f0dafdbe97afb261eab8244d7fd8b47e20ffcbcfee27fb168"

# Cleanup
oc delete namespace "$NAMESPACE"