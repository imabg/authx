FROM golang:1.26-alpine AS build

WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/authx ./cmd/authx

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/authx /app/authx
COPY --from=build /src/migrations /app/migrations
COPY config.example.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/authx"]
