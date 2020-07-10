FROM golang:1.14-alpine AS builder
WORKDIR /go/src/github.com/enova/scout
COPY . .
RUN apk add --no-cache git build-base
RUN go mod download
RUN go build -o scout ./

CMD ["./scout", "--help"]
