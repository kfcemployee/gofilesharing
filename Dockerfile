FROM golang:1.25.5-alpine as builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

ARG SERVICE_NAME    
RUN CGO_ENABLED=1 GOOS=linux go build -o /app-bin ./cmd/${SERVICE_NAME}/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat

WORKDIR /root/

COPY --from=builder /app-bin ./service

RUN mkdir -p /root/storage /root/tmp

CMD ["./service"]