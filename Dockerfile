FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o app ./cmd

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .

ENV TZ=Europe/Moscow

CMD ["./app"]
