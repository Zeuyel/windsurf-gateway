FROM node:22-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.1-alpine AS backend-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /windsurf-gateway ./cmd/

FROM alpine:3.22
RUN apk --no-cache add ca-certificates tzdata

ENV TZ=Asia/Shanghai
WORKDIR /app

COPY --from=backend-builder /windsurf-gateway ./windsurf-gateway
COPY --from=frontend-builder /app/web/dist ./web/dist
COPY .env.example ./.env.example

EXPOSE 8080
CMD ["./windsurf-gateway"]
