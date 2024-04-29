FROM golang:1.22.0 AS builder

WORKDIR /build
COPY . .
ENV CGO_ENABLED 0
RUN go build -a -trimpath -ldflags "-s -w" .

FROM scratch

COPY --from=builder /build/kor /usr/bin/kor
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT [ "/usr/bin/kor" ]
CMD ["--help"]
