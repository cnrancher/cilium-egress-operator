FROM registry.suse.com/bci/golang:1.24 AS operator-build

WORKDIR /app

# Pre-download dependencies
RUN --mount=type=cache,target=/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# Build the binary
ENV CGO_ENABLED=0
ENV GOCACHE=/root/.cache/go-build
ARG TAG=v0.1.0
ARG COMMIT=unknown
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=bind,source=.,target=. \
    go build \
        -buildmode=pie \
        -ldflags="-extldflags='-static' -s -w -X github.com/cnrancher/cilium-egress-operator/pkg/utils.Version=${TAG} -X github.com/cnrancher/cilium-egress-operator/pkg/utils.Commit=${COMMIT}" \
        -o /usr/local/bin/cilium-egress-operator \
        .

FROM registry.suse.com/bci/bci-base:16.0 AS operator

COPY --from=operator-build /usr/local/bin/ /usr/local/bin/

RUN useradd --uid 1007 cilium-egress-operator
ENV KUBECONFIG=/home/cilium-egress-operator/.kube/config
USER 1007

ARG TAG
LABEL org.opencontainers.image.source=https://github.com/cnrancher/cilium-egress-operator
LABEL org.opencontainers.image.description="Operator for Managing Cilium Egress Gateway Policies."
LABEL org.opencontainers.image.version="$TAG"

ENTRYPOINT [ "cilium-egress-operator" ]
