FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o random-api .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tini
WORKDIR /root/
COPY --from=builder /app/random-api .

EXPOSE 5003

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./random-api"]
