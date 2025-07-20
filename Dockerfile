# for build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bot ./cmd/bot

# for run
FROM alpine:latest

COPY --from=builder /app/bot /app/bot

CMD ["/app/bot"]