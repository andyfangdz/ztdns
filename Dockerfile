FROM golang:alpine AS builder
ADD https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR $GOPATH/src/github.com/andyfangdz/ztdns
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
RUN go build -o /ztdns .

FROM alpine
COPY --from=builder /ztdns ./
RUN apk add --no-cache ca-certificates && \
    update-ca-certificates
ENTRYPOINT ["./ztdns"]
EXPOSE 53
