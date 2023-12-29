FROM alpine

RUN apk update && \
    apk add --no-cache tzdata

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

COPY ./bld/tado-$TARGETOS-$TARGETARCH /app/tado

EXPOSE 8080
ENTRYPOINT ["/app/tado", "monitor"]
CMD []
