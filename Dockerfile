############################
# STEP 1 build executable binary
############################
FROM golang:1.23-alpine AS builder

RUN apk update && apk add --no-cache git make ca-certificates && update-ca-certificates

ARG SERVICE_NAME=ping-service

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=latest
ARG COMMIT_SHA=unknown

RUN export GOOS=linux
RUN export GOARCH=amd64

RUN make install

############################
# STEP 2 build a small image
############################
FROM alpine
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/eventhub_metrics /usr/local/bin/eventhub_metrics

RUN addgroup -S ehmetrics \
    && adduser -S ehmetrics -G ehmetrics \
    && chmod o+rx /usr/local/bin/eventhub_metrics \
    && apk upgrade --no-cache
USER ehmetrics

ENTRYPOINT ["eventhub_metrics"]
