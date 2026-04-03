FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/vpn-manager ./cmd/server
RUN go build -o /out/sysctl ./cmd/sysctl

FROM alpine:3.20
RUN adduser -D -g '' app
WORKDIR /app
COPY --from=builder /out/vpn-manager /usr/local/bin/vpn-manager
COPY --from=builder /out/sysctl /usr/local/bin/sysctl
COPY scripts ./scripts
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/vpn-manager"]
