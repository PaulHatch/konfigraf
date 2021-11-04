FROM ghcr.io/paulhatch/plgo:1.4.3 as builder

WORKDIR /usr/local/go/src/github.com/paulhatch/konfigraf
COPY . .

RUN plgo -x -v 1.0.0 .
RUN cat extension.sql >> ./build/konfigraf--1.0.0.sql

FROM postgres:13

RUN apt-get update && apt-get install make

COPY --from=builder /usr/local/go/src/github.com/paulhatch/konfigraf/build .

RUN make install