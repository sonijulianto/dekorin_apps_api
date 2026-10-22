FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server cmd/api/main.go

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Jakarta

COPY --from=builder /app/server /app/server
COPY --from=builder /app/web /app/web

RUN mkdir -p /app/uploads

EXPOSE 3000

CMD ["/app/server"]
