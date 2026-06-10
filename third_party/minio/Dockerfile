# Copyright 2019 The Kubeflow Authors
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

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d as downloader

COPY third_party /third_party

RUN apk add --no-cache curl

ENV OUTPUT_FOLDER /downloads/
RUN sh /third_party/download_source.sh </third_party/minio/repo-MPL.txt


# Minio image
FROM minio/minio:RELEASE.2019-08-14T20-37-41Z@sha256:b897f6c0bf815956a26bd7cb202c4b771d7c89c2e2a8edaa62f127cea5175522

# Copy concatenated license file
COPY third_party/minio/license.txt /third_party/license.txt

# Copy downloaded source code
COPY --from=downloader /downloads /third_party
