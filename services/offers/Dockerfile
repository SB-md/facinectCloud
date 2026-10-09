# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /out/offers ./cmd/offers

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/offers /app/offers
EXPOSE 8080
ENTRYPOINT ["/app/offers"]
