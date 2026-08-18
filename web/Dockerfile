ARG NODE_BUILD_IMAGE=node:22.23.2-alpine3.24
ARG NGINX_IMAGE=nginx:1.29.8-alpine3.23

FROM ${NODE_BUILD_IMAGE} AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

COPY . .
RUN npm run lint && npm run build

FROM ${NGINX_IMAGE}

COPY .docker-compose/nginx/conf.d/my.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=12 \
    CMD wget -qO- http://127.0.0.1:8080/healthz | grep -q ok || exit 1
