# renovate: datasource=docker depName=golang
FROM golang:1.26.10-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY api api
COPY cmd cmd
COPY internal internal
COPY pkg pkg
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -trimpath -o /out/netbird-operator cmd/main.go

FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.title="NetBird Operator" \
      org.opencontainers.image.source="https://github.com/SwanseaUniversityMedical/netbird-kubernetes-operator" \
      org.opencontainers.image.licenses="BSD-3-Clause"
COPY --from=build /out/netbird-operator /usr/local/bin/
USER 65532:65532
ENTRYPOINT ["netbird-operator"]
