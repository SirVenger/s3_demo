FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/storage ./cmd/storage

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/storage /app/storage
ENV DATA_DIR=/data \
    GC_TTL_HOURS=24 \
    GC_INTERVAL_MIN=30
VOLUME ["/data"]
EXPOSE 8081
ENTRYPOINT ["/app/storage"]
