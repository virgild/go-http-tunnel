FROM alpine:3.11.3

RUN apk --no-cache --update add ca-certificates

RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app && \
    mkdir -p /app

COPY build/tunneld-linux /app/tunneld
RUN chmod +x /app/tunneld && \
    chown -R app:app /app

USER root
WORKDIR /app

ENTRYPOINT ["/app/tunneld"]
