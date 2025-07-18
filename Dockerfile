# Dockerfile (at your project root)

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o newsapp ./cmd/ingestor/

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/newsapp .
EXPOSE 8080
ENTRYPOINT ["./newsapp"]

