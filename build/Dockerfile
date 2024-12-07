FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ENV VERSION=$VERSION

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-X main.version=$VERSION" \
    -o tado \
    tado.go

FROM alpine

RUN apk update && \
    apk add --no-cache tzdata

WORKDIR /app
COPY --from=builder /app/tado /app/tado

RUN /usr/sbin/addgroup app
RUN /usr/sbin/adduser app -G app -D
USER app

ENTRYPOINT ["/app/tado", "monitor"]
CMD []
