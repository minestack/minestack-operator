ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

ARG ARCH="amd64"
ARG OS="linux"
COPY bin/minestack-operator-${OS}-${ARCH} /bin/minestack-operator
USER 65532:65532

ENTRYPOINT [ "/bin/minestack-operator" ]
