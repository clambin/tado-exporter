FROM alpine

RUN apk update && \
    apk add --no-cache tzdata

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY ./bld/tado-monitor-$TARGETOS-$TARGETARCH /app/tado-monitor

EXPOSE 8080
ENTRYPOINT ["/app/tado-monitor"]
CMD []
