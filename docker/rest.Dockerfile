FROM golang:1.24 as build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/rest ./cmd/rest

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/rest /app/rest
COPY docker/rest-config.example.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/rest"]

