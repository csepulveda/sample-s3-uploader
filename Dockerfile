FROM golang:1.22-alpine3.20 as builder
WORKDIR /app
COPY . .
RUN 
RUN go build -ldflags "-s -w" -o main main.go

FROM alpine:3.20
RUN apk update
RUN apk upgrade --no-cache libcrypto3 libssl3 openssl
RUN apk --no-cache add ca-certificates
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/main .
USER app
ENTRYPOINT ["./main"]