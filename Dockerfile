FROM golang:1.10-alpine as builder
RUN apk add --update make git
WORKDIR src/git.containerum.net/ch/volume-manager
COPY . .
RUN VERSION=$(git describe --abbrev=0 --tags) make build-for-docker

FROM alpine:3.7
COPY --from=builder /tmp/volume-manager /app

ENV MODE="release" \
    LOG_LEVEL=4 \
    DB_USER="volume-manager" \
    DB_PASSWORD="vTsnHHnI" \
    DB_HOST="postgres:5432" \
    DB_SSLMODE="false" \
    DB_BASE="volume-manager" \
    LISTEN_ADDR=":4343" \
    BILLING_ADDR="billing-manager:5000" \
    KUBE_API_ADDR="kube-api:1214"

EXPOSE 4343

CMD "/volume-manager"
