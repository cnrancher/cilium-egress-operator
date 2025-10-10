FROM registry.suse.com/bci/bci-base:16.0 AS operator

ARG TARGETARCH
RUN zypper update -y && \
    zypper in -y -f vim && \
    zypper clean -a && \
    rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/*

RUN useradd --uid 1007 cilium-egress-operator
ENV KUBECONFIG=/home/cilium-egress-operator/.kube/config
COPY dist/cilium-egress-operator_linux_${TARGETARCH}*/ /usr/bin/
USER 1007
ENTRYPOINT [ "cilium-egress-operator" ]
