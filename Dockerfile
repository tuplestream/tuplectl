FROM debian:stable-slim AS release

WORKDIR /stage

RUN apt-get update
RUN apt-get -y install ca-certificates --no-install-recommends

ADD bin/tuplectl-linux-amd64 /usr/local/bin/tuplectl
