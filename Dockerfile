FROM golang:latest as builder

WORKDIR /
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/scout

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /go/bin/scout /usr/local/bin

CMD ["scout", "-h"]
