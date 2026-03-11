FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o openmanage .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/openmanage /usr/local/bin/openmanage
COPY --from=builder /app/templates /templates
EXPOSE 8080
CMD ["openmanage"]
