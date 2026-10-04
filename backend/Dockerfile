FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY ratelimiter ./ratelimiter
COPY backend ./backend
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend ./backend

FROM alpine:3.20
RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/backend /backend
EXPOSE 8080
ENTRYPOINT ["/backend"]
