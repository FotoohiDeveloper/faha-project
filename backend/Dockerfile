FROM golang:alpine AS builder

WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o faha_api ./cmd/api/main.go

FROM debian:bookworm-slim

RUN sed -i 's|http://deb.debian.org/debian|http://linux-mirror.liara.ir/repository/debian|g' /etc/apt/sources.list.d/debian.sources || true && \
    sed -i 's|http://deb.debian.org/debian-security|http://linux-mirror.liara.ir/repository/debian-security|g' /etc/apt/sources.list.d/debian.sources || true

RUN apt-get update && apt-get install -y ca-certificates curl tzdata && rm -rf /var/lib/apt/lists/*
ENV TZ=Asia/Tehran

WORKDIR /app
COPY --from=builder /app/faha_api .
COPY --from=builder /app/map.html .

EXPOSE 3000
CMD ["./faha_api"]