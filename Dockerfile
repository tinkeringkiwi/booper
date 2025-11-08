### Dockerfile for Booper
# Multi-stage build: compile Go binary, then produce a small runtime image

FROM golang:1.22-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src

# Use module-aware build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/booper ./

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/booper /app/booper
# Copy static frontend assets (templates, static files) so the server can serve them at runtime
COPY --from=build /src/frontend /app/frontend
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/booper"]
