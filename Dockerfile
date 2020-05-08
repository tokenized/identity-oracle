# Build Container
FROM golang:1.14-alpine3.11 AS build-env
RUN apk update && \
    apk add ca-certificates && \
    apk add linux-headers && \
    apk add gcc && \
    apk add libc-dev && \
    apk add --no-cache --upgrade git make ca-certificates && \
    apk add --no-cache python3 && \
    pip3 install --upgrade pip && \
    pip3 install boto3 && \
    rm -rf /var/cache/apk/*


COPY . /go/src/github.com/tokenized/identity-oracle
WORKDIR /go/src/github.com/tokenized/identity-oracle

ARG BUILD_ENV=prod
ARG DEPENDENCY_BRANCH=develop
ARG AWS_ACCESS_KEY_ID
ARG AWS_SECRET_ACCESS_KEY
ARG GITHUB_USER

ENV BUILD_ENV $BUILD_ENV
ENV AWS_ACCESS_KEY_ID $AWS_ACCESS_KEY_ID
ENV AWS_SECRET_ACCESS_KEY $AWS_SECRET_ACCESS_KEY
ENV GO111MODULE on


#RUN make deps
RUN go get ./...
RUN make dist 


# Final Container 
FROM alpine:3.11 AS oracled-env
RUN apk update && \
    apk add ca-certificates && \
    apk add --no-cache python3 && \
    pip3 install --upgrade pip && \
    pip3 install boto3 && \
    rm -rf /var/cache/apk/*

LABEL maintainer="Tokenized"
COPY --from=build-env /go/src/github.com/tokenized/identity-oracle/conf /srv/clients/conf
COPY --from=build-env /go/src/github.com/tokenized/identity-oracle/conf /conf
COPY --from=build-env /go/src/github.com/tokenized/identity-oracle/dist/identityoracled /srv/clients/bin/identityoracled

ENTRYPOINT source /srv/clients/conf/env.sh -e=$BUILD_ENV && "/srv/clients/bin/identityoracled"

# docker build -t identityoracled:latest --build-arg GITHUB_USER=<xyz>