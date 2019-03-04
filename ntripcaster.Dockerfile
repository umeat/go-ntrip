FROM golang:1.12.0-alpine as builder
WORKDIR /root/
ADD . .
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
RUN CGO_ENABLED=0 GOOS=linux go build -a -i -o ntripcaster ./cmd/ntripcaster/

FROM scratch
WORKDIR /root/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /root/ntripcaster .
ENTRYPOINT ["./ntripcaster"]
