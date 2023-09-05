FROM golang:1.21.0-alpine3.18 as builder

WORKDIR /app

COPY . .

RUN go mod tidy && \
    go build

FROM alpine:3.18 as runner

RUN apk add --no-cache curl

COPY --from=builder /app/kor /usr/bin/kor
