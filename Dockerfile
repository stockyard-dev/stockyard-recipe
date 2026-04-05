FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go mod download && CGO_ENABLED=0 go build -o recipe ./cmd/recipe/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/recipe .
ENV PORT=9805 DATA_DIR=/data
EXPOSE 9805
CMD ["./recipe"]
