FROM tuplestream/golang:latest AS build

WORKDIR /build

ADD . .

RUN ./build.bash -release

FROM debian:stable-slim AS release

WORKDIR /stage

COPY --from=build /build/bin/tuplectl-linux-amd64 /usr/local/bin/tuplectl
