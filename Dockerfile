FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Copy the source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the Go app for multiple architectures with the new binary name
# This will be handled by Docker BuildX during image build
RUN go build -o /cfddns ./cmd/main.go

# Use a small alpine image for the final container
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /cfddns .

# Command to run
ENTRYPOINT ["./cfddns"]