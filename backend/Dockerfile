FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy all source (including go.mod)
COPY . .

# Resolve dependencies and build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/backend .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /var/log/backend

COPY --from=builder /app/backend /app/backend

EXPOSE 8080

CMD ["/app/backend"]