# syntax=docker/dockerfile:1

FROM golang:1.25.2-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

RUN CGO_ENABLED=0 go build -o plucker-prod -ldflags="-w -s" .

FROM alpine:3.20

RUN apk add --no-cache yt-dlp ca-certificates

WORKDIR /app

COPY --from=builder /app/plucker-prod .

CMD ["./plucker-prod"]
