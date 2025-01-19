FROM golang:1.22-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./processor_service/cmd

FROM scratch
WORKDIR /app
COPY --from=builder /app/main /app/main
CMD ["/app/main"]