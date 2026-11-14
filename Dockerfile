FROM golang:1.27.3-alpine AS builder
WORKDIR /Src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/storage-init ./cmd/storage-init

FROM alpine:3.22
RUN apk add --no-cache ca-certificates ffmpeg && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/storage-init /app/storage-init
USER app
EXPOSE 8081
CMD ["/app/api"]
