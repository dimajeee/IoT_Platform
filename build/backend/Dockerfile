FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY . ./

RUN go build -o /out/backend ./cmd/backend

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/backend /app/backend

ENTRYPOINT ["/app/backend"]
