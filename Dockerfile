FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /windsurf-gateway ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /windsurf-gateway .
COPY .env.example .env.example

RUN mkdir -p /app/web/dist

EXPOSE 8080
CMD ["./windsurf-gateway"]
