FROM golang:1.16 as builder
WORKDIR /app
COPY go.mod go.mod
RUN go mod download
COPY app/ .
RUN GCO_ENABLED=1 GOOS=linux go build -tags netgo -a -installsuffix cgo -o /app/main .



FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/main /app/main

run apk add docker git

expose 8080
CMD ["/app/main"]
