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
#export API_URL="https://$(oc -n $NAMESPACE get route "ds-pipeline-$DSPA_NAME" -o jsonpath={.spec.host})"

# Start Tunnel
oc port-forward -n $NAMESPACE deployment/ds-pipeline-$DSPA_NAME "8888:8888" &

# Run Tests
cd $TEST_DIRECTORY
go run github.com/onsi/ginkgo/v2/ginkgo -r -v --cover -p --keep-going "$@" -- -namespace=$NAMESPACE -apiScheme=https -disableTlsCheck=true -serviceAccountName=pipeline-runner-$DSPA_NAME -repoName="opendatahub-io/data-science-pipelines"