FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# ponytail: limit build RAM untuk 8GB VPS (disable parallel compiler)
RUN CGO_ENABLED=0 GOMAXPROCS=1 go build -o /server ./cmd/server
RUN CGO_ENABLED=0 GOMAXPROCS=1 go build -o /seed ./db/seed
RUN CGO_ENABLED=0 GOMAXPROCS=1 go build -o /usertool ./cmd/usertool

# migrate CLI
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    poppler-utils tesseract-ocr tesseract-ocr-ind ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /server .
COPY --from=builder /seed .
COPY --from=builder /usertool .
COPY --from=builder /go/bin/migrate .
COPY db/migrations ./migrations
EXPOSE 8080
CMD ["./server"]
