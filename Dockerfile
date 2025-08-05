FROM --platform=$BUILDPLATFORM d3fk/tailwindcss:v3 AS tailwind

WORKDIR /workdir

COPY ./views/ /workdir/views/
COPY ./static/css/input.css /workdir/static/css/input.css

COPY ./tailwind.config.js /workdir/.

RUN [ "/tailwindcss", "-i", "./static/css/input.css", "-o", "./static/css/style.min.css", "--minify"]

# ---------- Stage 1: Build ----------
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 

# Install CA certs for later copying
RUN apk add --no-cache git ca-certificates

RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy go mod files and download deps
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN templ generate

COPY --from=tailwind /workdir/static/css/style.min.css ./static/css/style.min.css

ARG VERSION_TAG="dev"

# Build static Go binary
RUN go build -ldflags "-X github.com/oliverisaac/fanks/version.Tag=$VERSION_TAG" -o /fanks ./cmd/fanks/

# Create a minimal passwd file for non-root user (UID 10001)
RUN echo "nonroot:x:10001:10001:NonRoot User:/:/sbin/nologin" > /etc/passwd

# ---------- Stage 2: Final ----------
FROM alpine AS release

# Copy static binary
COPY --from=builder /fanks /fanks
COPY --from=builder /workdir/views /views

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy passwd file
COPY --from=builder /etc/passwd /etc/passwd

# Set non-root user
USER 10001

ENV PORT=:4000

# Expose application port
EXPOSE 4000

# Start the app
ENTRYPOINT ["/fanks"]

