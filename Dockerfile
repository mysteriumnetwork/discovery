FROM golang:1.18-alpine AS builder

RUN apk add build-base npm
WORKDIR /go/src/github.com/mysteriumnetwork/discovery
COPY go.mod go.sum ./
RUN go mod download

# Compile application
ADD . .
RUN go run mage.go -v build

FROM alpine:3

# Install packages
RUN apk add --no-cache ca-certificates git

# Install application
COPY --from=builder /go/src/github.com/mysteriumnetwork/discovery/build/discovery /usr/bin/discovery

EXPOSE 8080

CMD ["/usr/bin/discovery"]
