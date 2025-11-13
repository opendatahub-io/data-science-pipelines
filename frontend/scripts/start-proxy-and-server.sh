#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}
PORT_FORWARD=${PORT_FORWARD:-"true"}
ML_PIPELINE_SERVICE_PORT=${ML_PIPELINE_SERVICE_PORT:3002}
FRONTEND_SERVER_PORT=${FRONTEND_SERVER_PORT:-3001}

function clean_up() {
  set +e

  echo "Stopping background jobs..."
  # jobs -l
  kill -15 %1
  kill -15 %2
  kill -15 %3
}
trap clean_up EXIT SIGINT SIGTERM

echo "Preparing dev env for KFP frontend"

echo "Compiling node server..."
pushd server
npm run build
popd

# Frontend dev server proxies api requests to node server listening to
# localhost:${FRONTEND_SERVER_PORT} (default 3001, configurable via FRONTEND_SERVER_PORT env var).
# Note: The proxy field in frontend/package.json is set to localhost:3001 by default.
# If you change FRONTEND_SERVER_PORT, you may need to update package.json's proxy field accordingly.
#
# Node server proxies requests further to localhost:3002
# based on what request it is.
#
# localhost:3002 port forwards to ml_pipeline api server pod.

echo "Starting to port forward backend apis..."

if [ "${PORT_FORWARD}" = "true" ]; then
  kubectl port-forward -n $NAMESPACE svc/ml-pipeline 3002:8888 &
  kubectl port-forward -n $NAMESPACE svc/minio-service 9000:9000 &
fi

export MINIO_HOST=localhost
export MINIO_NAMESPACE=
export FRONTEND_SERVER_PORT
if [ "$1" == "--inspect" ]; then
  npm run mock:server:inspect $FRONTEND_SERVER_PORT
else
  npm run mock:server $FRONTEND_SERVER_PORT
fi
