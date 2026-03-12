FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init webhook-receiver && go build -o webhook-receiver .

FROM alpine:3.19
COPY --from=builder /app/webhook-receiver /usr/local/bin/webhook-receiver
EXPOSE 8088
ENTRYPOINT ["webhook-receiver"]
