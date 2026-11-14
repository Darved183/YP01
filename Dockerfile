# Сначала собирается программа. В итоговый образ попадает только она и файлы сайта.
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app ./cmd/app

FROM debian:bookworm-slim

WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/app /app/app
COPY frontend /app/frontend
COPY internal/database/migrations /app/internal/database/migrations
COPY docker-entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

ENV HOST=0.0.0.0
ENV PORT=8181
EXPOSE 8181
VOLUME /data

ENTRYPOINT ["/entrypoint.sh"]
