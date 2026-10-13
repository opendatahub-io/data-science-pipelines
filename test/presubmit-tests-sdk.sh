#!/bin/bash -ex
# Copyright 2020 Kubeflow Pipelines contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SETUP_ENV="${SETUP_ENV:-true}"
PYTEST_PARALLEL_WORKERS="${PYTEST_PARALLEL_WORKERS:-2}"
JUNIT_XML="${JUNIT_XML:-sdk-unit.xml}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Generate proto files (requires Docker)
  make -C sdk generate-python

<<<<<<< HEAD
  python3 -m pip install --upgrade pip
  python3 -m pip install -r sdk/python/requirements.txt 
  python3 -m pip install -r sdk/python/requirements-dev.txt
  python3 -m pip install setuptools
  python3 -m pip install wheel==0.42.0
  python3 -m pip install pytest-cov
  python3 -m pip install pytest-xdist
  python3 -m pip install pytest
  python3 -m pip install google_cloud_pipeline_components
  python3 -m pip install docker==7.1.0
  python3 -m pip install --upgrade protobuf
  python3 -m pip install sdk/python
  python3 -m pip install -e kubernetes_platform/python --force-reinstall
  python3 -m pip install -e backend/api/v2beta1/python_http_client --force-reinstall
=======
  # Sync all dependencies using uv (includes google_cloud_pipeline_components, docker)
  uv sync --extra ci
>>>>>>> upstream/master

  # Install workspace packages in editable mode
  uv pip install -e sdk/python
fi

runtime_dist_dir=$(mktemp -d)
trap 'rm -rf "$runtime_dist_dir"' EXIT
uv build --package kfp --wheel --out-dir "$runtime_dist_dir"
runtime_wheels=("$runtime_dist_dir"/kfp-*.whl)
if [[ ${#runtime_wheels[@]} -ne 1 || ! -f "${runtime_wheels[0]}" ]]; then
  echo "Expected exactly one freshly built kfp wheel in $runtime_dist_dir" >&2
  exit 1
fi

# Docker-backed cases need a source URL accessible inside their containers.
if [[ -z "${PULL_NUMBER}" ]]; then
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}#egg=kfp&subdirectory=sdk/python"
else
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@refs/pull/${PULL_NUMBER}/merge#egg=kfp&subdirectory=sdk/python"
fi

<<<<<<< HEAD
python -m pytest sdk/python/test -v -m regression --cov=kfp -n "${PYTEST_PARALLEL_WORKERS}" --junitxml="${JUNIT_XML}"
=======
# Keep pytest capture enabled: xdist cannot forward worker stdout/stderr with -s.
uv run python -m pytest sdk/python/test --ignore=sdk/python/test/runtime \
  -v -m regression --cov=kfp -n "${PYTEST_PARALLEL_WORKERS}"
>>>>>>> upstream/master

KFP_PACKAGE_PATH="${runtime_wheels[0]}" \
  uv run python -m pytest sdk/python/test/runtime -v -m regression \
    --cov=kfp --cov-append -n "${PYTEST_PARALLEL_WORKERS}"
