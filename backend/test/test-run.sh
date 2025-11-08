#!/bin/bash

set -o allexport
source .env
set +o allexport

# Create DSPA CR template
cat <<EOF > /tmp/dspa.yaml
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
oc apply -n $NAMESPACE -f /tmp/dspa.yaml
oc wait --for=condition=available deployment/ds-pipeline-$DSPA_NAME --timeout=20m

# Get API URL
export API_URL="https://$(oc -n $NAMESPACE get route "ds-pipeline-$DSPA_NAME" -o jsonpath={.spec.host})"

# Get API Token
SERVICE_ACCOUNT_NAME="ds-pipeline-$DSPA_NAME"
export API_TOKEN=$(oc create token "$SERVICE_ACCOUNT_NAME" --namespace "$NAMESPACE" --duration=60m)

# Run Tests
cd $TEST_DIRECTORY
go run github.com/onsi/ginkgo/v2/ginkgo -r -v --cover -p --keep-going "$@" -- -namespace=$NAMESPACE -apiUrl=$API_URL -authToken="$API_TOKEN" -disableTlsCheck=true -serviceAccountName=pipeline-runner-$DSPA_NAME -repoName="opendatahub-io/data-science-pipelines" -baseImage="registry.redhat.io/ubi9/python-312@sha256:e80ff3673c95b91f0dafdbe97afb261eab8244d7fd8b47e20ffcbcfee27fb168"