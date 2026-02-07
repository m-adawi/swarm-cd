# Stage 1: Build the frontend
FROM node:22-alpine3.19 AS frontend-build
WORKDIR /ui
COPY ui/package.json ui/package-lock.json ./
RUN npm install
COPY ui/ ./
RUN npm run build
# Fail this stage if tests fail
RUN npm run test

# Stage 2: Build the backend
FROM golang:1.22.5 AS backend-build
ARG TARGETARCH
WORKDIR /backend
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY util/ util/
COPY web/ web/
COPY swarmcd/ swarmcd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o /swarm-cd ./cmd/
RUN go test ./swarmcd/

# Stage 3: Final production image (depends on previous stages)
FROM alpine:3.22.1
WORKDIR /app
RUN apk add --no-cache ca-certificates gnupg && update-ca-certificates
# Copy the built backend binary from the backend build stage
COPY --from=backend-build /swarm-cd /app/
# Copy the built frontend from the frontend build stage
COPY --from=frontend-build /ui/dist/ /app/ui/
# Sets the web server mode to release
ENV GIN_MODE=release
# Set the entry point for the application
COPY entrypoint.sh /
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/swarm-cd"]
