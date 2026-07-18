FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o app ./cmd

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY russian-trusted-root-ca.crt /usr/local/share/ca-certificates/russian-trusted-root-ca.crt

RUN update-ca-certificates

WORKDIR /app

COPY --from=builder /app/app .

ENV TZ=Europe/Moscow

CMD ["./app"]