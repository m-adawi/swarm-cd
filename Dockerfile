FROM node:22-alpine3.19 AS ui
WORKDIR /ui
COPY ui/package.json ui/package-lock.json .
RUN npm install
COPY ui/ .
RUN npm run build

FROM golang:1.22.5 AS backend
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
COPY --from=ui /ui/dist/ .
COPY --from=backend /swarm-cd .
CMD ["/app/swarm-cd"]