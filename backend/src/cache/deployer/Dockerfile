FROM gcr.io/google.com/cloudsdktool/google-cloud-cli:alpine@sha256:5d433d2e7bba6bb4710f58280681effd190720c57156a582e7ed905e7a57c65c

RUN apk add --update \
    curl \
    jq \
    openssl

RUN gcloud components install kubectl

ADD backend/src/cache/deployer/* /kfp/cache/deployer/
RUN chmod -R 777 /kfp/cache/deployer

WORKDIR /kfp/cache/deployer

RUN chmod +x deploy-cache-service.sh
RUN chmod +x webhook-create-signed-cert.sh
RUN chmod +x webhook-patch-ca-bundle.sh

COPY third_party/license.txt /third_party/license.txt

ENTRYPOINT ["/bin/sh", "/kfp/cache/deployer/deploy-cache-service.sh"]
