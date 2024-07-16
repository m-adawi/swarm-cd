FROM golang:1.22.5 AS base

COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY util/ util/
COPY web/  web/ 
COPY swarmcd/ swarmcd/
RUN CGO_ENABLED=0 GOOS=linux go build -o /swarm-cd ./cmd/

FROM alpine:3.2
WORKDIR /app
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY --from=base /swarm-cd .
CMD ["/app/swarm-cd"]