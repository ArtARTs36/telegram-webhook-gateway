# syntax=docker/dockerfile:1

FROM golang:1.26.0-alpine AS builder

ARG APP_VERSION="undefined"
ARG BUILD_TIME="undefined"

WORKDIR /go/src/github.com/artarts36/telegram-webhook-gateway

RUN apk add git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X 'main.Version=${APP_VERSION}' -X 'main.BuildDate=${BUILD_TIME}'" -o /go/bin/telegram-webhook-gateway /go/src/github.com/artarts36/telegram-webhook-gateway/cmd/telegram-webhook-gateway/main.go

######################################################

FROM alpine:3.20

COPY --from=builder /go/bin/telegram-webhook-gateway /go/bin/telegram-webhook-gateway

ENTRYPOINT ["/go/bin/telegram-webhook-gateway"]
