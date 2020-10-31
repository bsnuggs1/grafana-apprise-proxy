FROM golang:alpine AS builder

WORKDIR $GOPATH/src/grafana-apprise-proxy/.
COPY . .

RUN apk add --no-cache git mercurial \
    && go get -d -v \
    && apk del git mercurial

RUN go build -o /go/bin/grafana-apprise-proxy

FROM alpine:3.7

COPY --from=builder /go/bin/grafana-apprise-proxy /

ENV GRAFANA_APPRISE_PROXY_TARGET_PORT 80

# RUN chmod +x /go/bin/grafana-apprise-proxy
CMD ["/grafana-apprise-proxy"]