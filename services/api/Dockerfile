# Build context is the repository root: the Go module lives there.
# Targets: api (HTTP server), worker (background jobs), migrate (schema only, never seeds).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./services/api/cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./services/api/cmd/worker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./services/api/cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS api
WORKDIR /app
COPY --from=build /out/api /app/api
# Local storage writes here; mount a persistent volume in production.
ENV MEDIA_LOCAL_DIR=/app/data/media
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
WORKDIR /app
COPY --from=build /out/worker /app/worker
USER nonroot
ENTRYPOINT ["/app/worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS migrate
WORKDIR /app
COPY --from=build /out/migrate /app/migrate
COPY services/api/migrations /app/migrations
ENV MIGRATIONS_DIR=/app/migrations
USER nonroot
ENTRYPOINT ["/app/migrate"]
