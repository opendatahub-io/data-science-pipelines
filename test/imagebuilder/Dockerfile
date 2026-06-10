# This image contains script to pull github code, build image and push to gcr
# Available at gcr.io/ml-pipeline/image-builder

FROM google/cloud-sdk:279.0.0@sha256:ce27a40af01cefc3f6a1f2d05ad036364a521d0b61dcc5273f2516347792977f

COPY ./build.sh /build.sh
RUN chmod +x /build.sh

ENTRYPOINT ["/build.sh"]
