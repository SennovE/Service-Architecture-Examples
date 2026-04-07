FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o myservice ./

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/myservice .
COPY --from=builder /app/api ./api
CMD ["./myservice"]