FROM golang:1.24.4 AS builder
WORKDIR /app
COPY . .
RUN go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o user-service

# Use small distroless for static binaries
FROM gcr.io/distroless/static
WORKDIR /
COPY --from=builder /app/user-service .
EXPOSE 8080
ENTRYPOINT ["/user-service"]
