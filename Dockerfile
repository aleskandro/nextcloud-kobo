FROM golang:1.23 AS builder

WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOARCH=arm go build -a -o manager main.go

FROM scratch

WORKDIR /etc/ssl/certs/
COPY --from=gcr.io/distroless/base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /usr/local/nextcloud-kobo/
COPY --from=builder /go/src/app/manager ./nextcloud-kobo

COPY /root/ /
ENTRYPOINT ["/usr/local/nextcloud-kobo/nextcloud-kobo"]