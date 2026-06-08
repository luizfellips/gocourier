FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /scheduler ./cmd/scheduler

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app

COPY --from=builder /api /app/api
COPY --from=builder /worker /app/worker
COPY --from=builder /scheduler /app/scheduler
