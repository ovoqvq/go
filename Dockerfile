FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
# 如果有配置文件则拷贝：COPY --from=builder /app/config.yaml .
EXPOSE 8081
CMD ["./server"]