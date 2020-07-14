FROM ubuntu:18.04 as build-base

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    wget \
    curl \
    ca-certificates \
    g++ \
    gcc \
    libc6-dev \
    make \
    pkg-config \
    gnupg

# Install PostgreSQL 12 development tools
RUN echo "deb http://apt.postgresql.org/pub/repos/apt bionic-pgdg main" >> /etc/apt/sources.list.d/pgdg.list && \
    curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
    apt-get update && \
    apt-get -y install postgresql-server-dev-12 && \
    rm -rf /var/lib/apt/lists/*

ENV GOLANG_VERSION 1.13.7

# Install Go
RUN wget -O go.tgz "https://golang.org/dl/go1.13.7.linux-amd64.tar.gz"; \
    tar -C /usr/local -xzf go.tgz; \
    rm go.tgz; \
    export PATH="/usr/local/go/bin:$PATH"; \
    go version;

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Install PLGO
RUN go get -u github.com/paulhatch/plgo/plgo

WORKDIR $GOPATH

FROM build-base  as builder

COPY . /go/src/github.com/paulhatch/konfigraf/
COPY docker-init.sh .

CMD bash docker-init.sh