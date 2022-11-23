FROM golang:1.19-alpine3.16 AS build

WORKDIR /app

COPY . .
RUN go build ./cmd/graph-intel-api

FROM alpine:3.16

WORKDIR /app

COPY --from=build /app/graph-intel-api graph-intel-api

ENTRYPOINT ["/app/graph-intel-api"]
