FROM debian:stable-slim AS release

WORKDIR /stage

RUN apt-get update && apt-get -y install ca-certificates --no-install-recommends && apt-get clean && rm -rf /var/lib/apt/lists/*

COPY bin/tuplectl-linux-amd64 /usr/local/bin/tuplectl
