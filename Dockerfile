FROM golang:1.26.1-alpine3.23 AS build

WORKDIR /src

ARG APP_PATH=./cmd/server
ARG BIN_NAME=app

COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY config ./config
COPY core ./core
COPY http ./http
COPY pool ./pool
COPY security ./security

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/${BIN_NAME} ${APP_PATH}

FROM alpine:3.23

RUN addgroup -S mtws && adduser -S -G mtws mtws

WORKDIR /app

ARG BIN_NAME=app

COPY --from=build /out/${BIN_NAME} /usr/local/bin/app
COPY security/waf/default-policy.txt /etc/mtws/waf-policy.txt

EXPOSE 8080

USER mtws

ENTRYPOINT ["/usr/local/bin/app"]
