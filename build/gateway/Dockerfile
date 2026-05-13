FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY . ./

RUN go build -o /out/gateway ./cmd/gateway

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/gateway /app/gateway

ENTRYPOINT ["/app/gateway"]
