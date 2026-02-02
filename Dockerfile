# Stage 1: Builder
FROM golang:1.25-alpine AS builder

# Install git if needed (not needed for modernc sqlite usually, but safely)
RUN apk add --no-cache git

WORKDIR /app

# Copy Mod files first for cache
COPY go.mod go.sum ./
RUN go mod download

# Copy Source
COPY . .

# Build Static Binary
# -s -w: Strip debug info to reduce size
# CGO_ENABLED=0: Static binary for scratch/alpine
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o hvac-app main.go

# Stage 2: Runner
FROM alpine:latest

# Install basic certs and timezone data
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy Binary
COPY --from=builder /app/hvac-app .

# Copy Static Assets & Views (Required for UI)
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/views ./views

# [OPTIONAL] Copy FCM Key if exists (or user mounts it)
# COPY --from=builder /app/serviceAccountKey.json .

# Create Data Directory
RUN mkdir /pb_data

# Expose Port
EXPOSE 8090

# Volume
VOLUME ["/pb_data"]

# Command
CMD ["/app/hvac-app", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]
