# Note: this is cross-building for armv7l, which is the architecture of the Kobo Elipsa 2E devices
# This Dockerfile is meant to be used only to support the build of the KoboRoot.tgz
# The final image is based on scratch, with the binary, the ca-certificates and any other content to ship to the device.
# The image is squashed and stored as KoboRoot.tgz for the release.
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