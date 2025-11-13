#!/bin/bash
set -e

# This script generates TypeScript clients from Swagger definitions
# for both v1beta1 and v2beta1 APIs
#
# This script can be run either:
# 1. Inside the kfp-api-generator container (recommended) - via `make generate-swagger-clients` from frontend/
# 2. Locally with swagger-codegen-cli.jar installed (legacy)

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
FRONTEND_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_API_DIR="$FRONTEND_DIR/../backend/api"
SWAGGER_CONFIG="$FRONTEND_DIR/swagger-config.json"

# Determine which swagger-codegen-cli to use
# If running in container, use the containerized version
if [ -f "/usr/bin/swagger-codegen-cli.jar" ]; then
  SWAGGER_CODEGEN_JAR="/usr/bin/swagger-codegen-cli.jar"
  echo "Using containerized swagger-codegen-cli"
else
  # Fall back to local version for backward compatibility
  SWAGGER_CODEGEN_JAR="$FRONTEND_DIR/swagger-codegen-cli.jar"
  echo "Using local swagger-codegen-cli"

  # Check if swagger-codegen-cli.jar exists locally
  if [ ! -f "$SWAGGER_CODEGEN_JAR" ]; then
    echo "ERROR: swagger-codegen-cli.jar not found at $SWAGGER_CODEGEN_JAR"
    echo ""
    echo "Please use the containerized approach (recommended):"
    echo "  cd frontend && make generate-swagger-clients"
    echo ""
    echo "Or download swagger-codegen-cli.jar for local use:"
    echo "  curl -o $SWAGGER_CODEGEN_JAR https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.7/swagger-codegen-cli-2.4.7.jar"
    exit 1
  fi
fi

echo "Generating v1beta1 API clients..."

# v1beta1 APIs (client-facing)
V1_APIS=("experiment" "job" "pipeline" "run" "filter" "visualization")
for api in "${V1_APIS[@]}"; do
  echo "  Generating v1beta1/$api..."
  java -jar "$SWAGGER_CODEGEN_JAR" generate \
    -i "$BACKEND_API_DIR/v1beta1/swagger/${api}.swagger.json" \
    -l typescript-fetch \
    -o "$FRONTEND_DIR/src/apis/$api" \
    -c "$SWAGGER_CONFIG"
done

# v1beta1 auth (server-side)
echo "  Generating v1beta1/auth..."
java -jar "$SWAGGER_CODEGEN_JAR" generate \
  -i "$BACKEND_API_DIR/v1beta1/swagger/auth.swagger.json" \
  -l typescript-fetch \
  -o "$FRONTEND_DIR/server/src/generated/apis/auth" \
  -c "$SWAGGER_CONFIG"

echo "Generating v2beta1 API clients..."

# v2beta1 APIs (client-facing)
V2_APIS=("artifact" "experiment" "pipeline" "run" "filter" "visualization")
for api in "${V2_APIS[@]}"; do
  echo "  Generating v2beta1/$api..."
  java -jar "$SWAGGER_CODEGEN_JAR" generate \
    -i "$BACKEND_API_DIR/v2beta1/swagger/${api}.swagger.json" \
    -l typescript-fetch \
    -o "$FRONTEND_DIR/src/apisv2beta1/$api" \
    -c "$SWAGGER_CONFIG"
done

# v2beta1 recurring_run (special naming)
echo "  Generating v2beta1/recurring_run..."
java -jar "$SWAGGER_CODEGEN_JAR" generate \
  -i "$BACKEND_API_DIR/v2beta1/swagger/recurring_run.swagger.json" \
  -l typescript-fetch \
  -o "$FRONTEND_DIR/src/apisv2beta1/recurringrun" \
  -c "$SWAGGER_CONFIG"

# v2beta1 auth (server-side)
echo "  Generating v2beta1/auth..."
java -jar "$SWAGGER_CODEGEN_JAR" generate \
  -i "$BACKEND_API_DIR/v2beta1/swagger/auth.swagger.json" \
  -l typescript-fetch \
  -o "$FRONTEND_DIR/server/src/generated/apisv2beta1/auth" \
  -c "$SWAGGER_CONFIG"

echo "Swagger client generation complete!"
