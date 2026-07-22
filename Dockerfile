FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/server ./cmd/main

FROM alpine:3.21

RUN adduser -D -u 1000 app
USER app

COPY --from=build /bin/server /bin/server

EXPOSE 8080
ENTRYPOINT ["/bin/server"]
