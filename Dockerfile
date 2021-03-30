FROM golang:1.16-alpine AS builder

WORKDIR /go/src/github.com/mysteriumnetwork/ndiscovery
COPY go.mod go.sum ./
RUN go mod download

# Compile application
ADD . .
RUN go run mage.go -v build

FROM alpine:3

# Install packages
RUN apk add --no-cache ca-certificates git

# Install application
COPY --from=builder /go/src/github.com/mysteriumnetwork/ndiscovery/build/ndiscovery /usr/bin/ndiscovery

EXPOSE 3000

CMD ["/usr/bin/discovery"]
