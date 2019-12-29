# Build Container
FROM golang:1.11-alpine3.8 AS build-env
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
ENV GITHUB_USER $GITHUB_USER

# this is required so we can clone nexus-api , which is private
RUN git config --global --add url."https://$GITHUB_USER:@github.com/".insteadOf "https://github.com/"

RUN git clone --single-branch --branch $DEPENDENCY_BRANCH https://github.com/tokenized/specification.git ../specification && \
    git clone --single-branch --branch $DEPENDENCY_BRANCH https://github.com/tokenized/envelope.git ../envelope && \
    git clone --single-branch --branch $DEPENDENCY_BRANCH https://github.com/tokenized/smart-contract.git ../smart-contract 
    
RUN git clone --single-branch --branch $DEPENDENCY_BRANCH https://github.com/tokenized/nexus-api.git ../nexus-api

RUN make deps

RUN make dist 


# Final Container 
FROM alpine:3.8 AS oracled-env
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