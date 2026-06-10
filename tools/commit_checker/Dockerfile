FROM registry.access.redhat.com/ubi9/go-toolset:1.22@sha256:e4193e71ea9f2e2504f6b4ee93cadef0fe5d7b37bba57484f4d4229801a7c063 as builder

WORKDIR /tmp
RUN git clone https://github.com/openshift/build-machinery-go.git && \
    cd /tmp/build-machinery-go/commitchecker && \
    go build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5@sha256:a50731d3397a4ee28583f1699842183d4d24fadcc565c4688487af9ee4e13a44

WORKDIR /bin

COPY --from=builder /tmp/build-machinery-go/commitchecker/commitchecker /bin/commitchecker
RUN microdnf install git && \
    chmod +x /bin/commitchecker && \
    mkdir -p /src/app-root

WORKDIR /src/app-root
ENTRYPOINT ["commitchecker"]

VOLUME ["/src/app-root"]

LABEL name="Commit Checker tool" \
      summary="commitchecker validates a range of commits in a git repository and ensures they meet specific requirements: \
        1. The author's email address does not start with "root@". \
        2. The message starts with one of: \
            i. UPSTREAM: <PR number|carry|drop>: description \
            ii. UPSTREAM: revert: \
        This is useful for repositories that are downstream forks of upstream repositories."
