# Copyright 2021-2024 The Kubeflow Authors
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

# Build arguments
ARG SOURCE_CODE=.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BUILDER_ARCH=multiarch

FROM registry.access.redhat.com/ubi9/go-toolset:1.26.3@sha256:45bd06d392349dfdb09073514cfebcc5082db33301347b51dcae8e5516a89126 AS go-toolset-multiarch
FROM registry.access.redhat.com/ubi9/go-toolset:1.26.3@sha256:3880436381044c6d89c2311f2a7dc8d64224668bf84b43810982b5a9ddd851d0 AS go-toolset-arm64
FROM go-toolset-${BUILDER_ARCH} AS builder

ARG SOURCE_CODE
ARG TARGETOS
ARG TARGETARCH

USER root

# Zig is used as a C cross-compiler for CGO (required by mattn/go-sqlite3).
# Unlike gcc, Zig can target any architecture via --target without needing
# separate cross-toolchains or a cross-linker.
ENV ZIG_VERSION=0.16.0
ENV ZIG_SHA256_x86_64=70e49664a74374b48b51e6f3fdfbf437f6395d42509050588bd49abe52ba3d00
ENV ZIG_SHA256_aarch64=ea4b09bfb22ec6f6c6ceac57ab63efb6b46e17ab08d21f69f3a48b38e1534f17
RUN ARCH=$(uname -m) && \
    eval "EXPECTED=\$ZIG_SHA256_${ARCH}" && \
    curl --fail -sL "https://ziglang.org/download/${ZIG_VERSION}/zig-${ARCH}-linux-${ZIG_VERSION}.tar.xz" -o /tmp/zig.tar.xz && \
    echo "${EXPECTED}  /tmp/zig.tar.xz" | sha256sum -c - && \
    tar -xJ -C /usr/local -f /tmp/zig.tar.xz && \
    rm /tmp/zig.tar.xz && \
    ln -s /usr/local/zig-${ARCH}-linux-${ZIG_VERSION}/zig /usr/local/bin/zig

COPY ${SOURCE_CODE}/go.mod ./
COPY ${SOURCE_CODE}/go.sum ./
COPY ${SOURCE_CODE}/kubernetes_platform/go.mod ./kubernetes_platform/go.mod
COPY ${SOURCE_CODE}/api/go.mod ./api/go.mod

RUN GOTOOLCHAIN=local GO111MODULE=on go mod download

# Copy the source
COPY ${SOURCE_CODE}/ ./

RUN case "${TARGETARCH}" in \
      amd64) ZIG_TARGET="x86_64-linux-gnu" ;; \
      arm64) ZIG_TARGET="aarch64-linux-gnu" ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    CC="zig cc --target=${ZIG_TARGET}" && \
    GOTOOLCHAIN=local GO111MODULE=on CGO_ENABLED=1 CC="${CC}" GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
      GOFIPS140=v1.0.0 \
      go build -tags no_openssl -o /bin/apiserver ./backend/src/apiserver/

# 2. Compile preloaded pipeline samples
FROM registry.access.redhat.com/ubi9/python-311@sha256:7b6cb58d3ff034df7b300800bd89a469d9bd2f739d43250d76b9c9e805307ab5 AS python-311-multiarch
FROM registry.access.redhat.com/ubi9/python-311@sha256:c7622eed60c05b1f75a0d2f71b265148c2e07c8a53a7a4bc6ebdd7fbb2168da3 AS python-311-arm64
FROM python-311-${BUILDER_ARCH} AS compiler

ARG TARGETOS TARGETARCH

USER root
RUN dnf install -y python3-setuptools jq
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py
COPY backend/requirements.txt .
RUN python3 -m pip install -r requirements.txt --no-cache-dir

# Downloading Argo CLI so that the samples are validated
ENV ARGO_VERSION=v3.6.7
RUN curl -sLO https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-${TARGETOS:-linux}-${TARGETARCH:-amd64}.gz && \
    gunzip argo-${TARGETOS:-linux}-${TARGETARCH:-amd64}.gz && \
    chmod +x argo-${TARGETOS:-linux}-${TARGETARCH:-amd64} && \
    mv ./argo-${TARGETOS:-linux}-${TARGETARCH:-amd64} /usr/local/bin/argo

WORKDIR /
COPY ./samples /samples
COPY backend/src/apiserver/config/sample_config.json /samples/

# Compiling the preloaded samples.
# The default image is replaced with the GCR-hosted python image.
RUN set -e; \
    < /samples/sample_config.json jq ".pipelines[].file" --raw-output | while read pipeline_yaml; do \
    pipeline_py="${pipeline_yaml%.yaml}"; \
    echo "Compiling: \"$pipeline_py\"" && python3 "$pipeline_py" && echo -n "Output: " && ls "$pipeline_py.yaml"; \
    done

# 3. Install runtime dependencies for the target architecture
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1784705586@sha256:5a68031c6c8f0af8da986896353ca62a78576fe70e8d0334733e25248df5cbde AS ubi-minimal-amd64
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1784705586@sha256:8673499787c3fed56e5da849e4060797c2cc5f05219ab804b40f0d6c0acf0ebf AS ubi-minimal-arm64
FROM ubi-minimal-${TARGETARCH} AS runtime-deps
RUN microdnf install -y ca-certificates wget && microdnf clean all

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1784705586

WORKDIR /bin

COPY --from=builder /opt/app-root/src/backend/src/apiserver/config/ /config
COPY --from=builder /bin/apiserver /bin/apiserver

COPY --from=compiler /samples/ /samples/
COPY --from=runtime-deps /etc/pki /etc/pki
COPY --from=runtime-deps /usr/bin/wget /usr/bin/wget

USER 1001

# Expose apiserver port
EXPOSE 8888

# Start the apiserver
CMD ["/bin/sh", "-c", "/bin/apiserver --config=/config --sampleconfig=/config/sample_config.json -logtostderr=true --logLevel=${LOG_LEVEL}"]
