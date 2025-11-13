# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Makefile for frontend code generation

# Use the same image as backend API generation
PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:master
IMAGE_TAG=kfp-api-generator
CONTAINER_ENGINE ?= docker

# Generate TypeScript clients from Swagger definitions using containerized swagger-codegen-cli
# This eliminates the need to install Java and swagger-codegen-cli.jar locally
.PHONY: generate-swagger-clients
generate-swagger-clients:
	@echo "Generating TypeScript clients from Swagger definitions..."
	${CONTAINER_ENGINE} run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-v "$$(pwd)/..":/go/src/github.com/kubeflow/pipelines:Z \
		$(PREBUILT_REMOTE_IMAGE) \
		/go/src/github.com/kubeflow/pipelines/frontend/scripts/generate-swagger-clients.sh

# Generate TypeScript clients using locally built image (for development)
.PHONY: generate-swagger-clients-from-scratch
generate-swagger-clients-from-scratch:
	@echo "Building kfp-api-generator image locally..."
	cd ../backend/api && make image
	@echo "Generating TypeScript clients from Swagger definitions..."
	${CONTAINER_ENGINE} run --interactive --rm \
		--user $$(id -u):$$(id -g) \
		-v "$$(pwd)/..":/go/src/github.com/kubeflow/pipelines:Z \
		$(IMAGE_TAG) \
		/go/src/github.com/kubeflow/pipelines/frontend/scripts/generate-swagger-clients.sh

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  generate-swagger-clients                - Generate TypeScript clients using pre-built container image (recommended)"
	@echo "  generate-swagger-clients-from-scratch   - Generate TypeScript clients using locally built container image"
	@echo "  help                                    - Show this help message"
