FROM golang:1.22.5 AS base

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /swarm-cd

FROM alpine:3.2
WORKDIR /app
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY --from=base /swarm-cd .
CMD ["/app/swarm-cd"]