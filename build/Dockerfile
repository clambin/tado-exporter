FROM golang:1.15 AS builder

WORKDIR /build

COPY . ./

ARG BUILD_VERSION=development

RUN CGO_ENABLED=0 go build -ldflags "-X github.com/clambin/tado-exporter/internal/version.BuildVersion=$BUILD_VERSION" tado-monitor.go

FROM alpine

WORKDIR /app

COPY --from=builder /build/tado-monitor /app

EXPOSE 8080
ENTRYPOINT ["/app/tado-monitor"]
CMD []
