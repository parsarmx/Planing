FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS api
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /poker .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S poker && adduser -S poker -G poker && mkdir -p /app/data && chown poker:poker /app/data
WORKDIR /app
COPY --from=api /poker /app/poker
COPY backend/config/config.yml /app/config/config.yml
COPY --from=web /web/dist /app/web
ENV APP_ADDR=0.0.0.0:8080 FRONTEND_PATH=/app/web
USER poker
EXPOSE 8080
CMD ["/app/poker", "serve"]
