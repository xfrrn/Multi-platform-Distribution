FROM node:22-alpine AS web-builder
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/update-server ./cmd/server

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /out/update-server /app/update-server
COPY --from=web-builder /src/web/dist /app/web
ENV WEB_DIST_DIR=/app/web
EXPOSE 8080
ENTRYPOINT ["/app/update-server"]
