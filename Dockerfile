# Build the application from source
FROM golang:1.24-bookworm AS builder

WORKDIR /src

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . .
RUN make build


# Deploy the application binary into a lean image
FROM alpine

WORKDIR /app
RUN mkdir -p /app/logs

RUN apk add curl

# Copy the binary to the production image from the builder stage.
COPY --from=builder /src/script/entrypoint.sh /src/bin/bifrost /src/config/docker/config.yaml /app/
RUN chmod +x /app/entrypoint.sh

# Run the web service on container startup.
CMD ["/app/entrypoint.sh"]
