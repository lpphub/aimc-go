FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download || go mod tidy

COPY . .
RUN go build -o /app/server .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server ./
ADD config ./config

RUN apk update && apk add --no-cache tzdata
ENV TZ=Asia/Shanghai

CMD ["/app/server"]