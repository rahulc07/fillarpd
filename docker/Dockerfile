# builder
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache make git

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make

# runner
FROM alpine:latest
RUN apk add --no-cache ca-certificates iproute2 libpcap libbpf
WORKDIR /root/
COPY --from=builder /app/fillarpd .
ENTRYPOINT ["./fillarpd"]