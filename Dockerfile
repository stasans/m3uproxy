# Stage 1: Build the Go binary
FROM golang:1.22.3-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the module dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .
RUN chmod +x scripts/*

RUN go build -o m3uproxy

FROM alpine:latest

COPY --from=builder /app/m3uproxy /app/m3uproxy
COPY --from=builder /app/conf /app/conf
COPY --from=builder /app/assets /app/assets
COPY --from=builder /app/scripts/entrypoint.sh /app/entrypoint.sh

RUN apk --no-cache add ca-certificates ffmpeg bash && \
    mkdir -p /app/cache 

WORKDIR /app

EXPOSE 8080

CMD ["/app/entrypoint.sh"]
