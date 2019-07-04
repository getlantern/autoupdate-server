FROM golang:1.12 as builder

WORKDIR $GOPATH/src/github.com/getlantern/autoupdate-server
# Copying the mod and sum files, then running 'go mod download' creates a
# separate layer of cached dependencies.
COPY go.mod go.sum ./
RUN GO111MODULE=on go mod download
COPY . .
RUN GO111MODULE=on go build -o /bin/autoupdate-server
RUN GO111MODULE=on go build -o /bin/autoupdate-server-mock -tags mock


# Running container
FROM debian:jessie

RUN apt-get update && apt-get install -y \
	ca-certificates

COPY --from=builder /bin/autoupdate-server /bin/autoupdate-server
COPY --from=builder /bin/autoupdate-server-mock /bin/autoupdate-server-mock

COPY bin/entrypoint.sh /bin/entrypoint.sh

RUN mkdir /app
VOLUME [ "/keys", "/app" ]

ENTRYPOINT ["/bin/entrypoint.sh"]
