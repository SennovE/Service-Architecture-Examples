FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

COPY . .

RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o myservice ./cmd/producer

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache curl

COPY --from=builder /app/myservice .
COPY --from=builder /app/api ./api

CMD ["./myservice"]