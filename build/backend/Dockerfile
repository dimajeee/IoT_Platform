FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /usr/local/bin/iot-backend ./cmd/backend

EXPOSE 8080

ENTRYPOINT ["iot-backend"]
