FROM node:22-alpine AS builder
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web ./
RUN npm run build

FROM nginx:1.27-alpine
COPY deploy/web.nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/web/dist /usr/share/nginx/html
EXPOSE 80
