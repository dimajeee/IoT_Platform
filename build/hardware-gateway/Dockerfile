FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY . ./

RUN go build -o /out/hardware-gateway ./cmd/hardware-gateway

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/hardware-gateway /app/hardware-gateway

ENTRYPOINT ["/app/hardware-gateway"]
