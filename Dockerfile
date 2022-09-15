# Reference: https://docs.docker.com/language/golang/build-images/

# Start from golang base image
FROM golang:latest as builder

# Set the Current Working Directory inside the container
WORKDIR /

# Copy module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build the Go app
COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/scout .

######## Start a new stage from scratch #######
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /go/bin/scout /usr/local/bin

CMD ["scout", "-h"]
