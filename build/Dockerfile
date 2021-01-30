FROM alpine

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder/bld/tado-monitor-$TARGETOS-$TARGETARCH /build/tado-monitor /app/tado-monitor

EXPOSE 8080
ENTRYPOINT ["/app/tado-monitor"]
CMD []
