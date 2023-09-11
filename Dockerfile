FROM golang:1.20.2-alpine AS builder

WORKDIR /build
COPY . .
ENV CGO_ENABLED 0
RUN go build .

FROM alpine:3.18

COPY --from=builder /build/kor /kor
ENTRYPOINT [ "/kor" ]
CMD ["--help"]