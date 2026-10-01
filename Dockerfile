FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-star .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder /out/go-star /app/go-star
COPY init/ip2region.xdb /app/init/ip2region.xdb
EXPOSE 8080
ENTRYPOINT ["/app/go-star", "-f", "/app/settings.docker.yaml"]