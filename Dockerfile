FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd/backend ./cmd/backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend ./cmd/backend

FROM alpine:3.21
RUN adduser -D -H -u 10001 app
COPY --from=builder /out/backend /usr/local/bin/backend
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/backend"]
