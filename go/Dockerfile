FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go/go.mod ./
COPY go/ratelimiter ./ratelimiter
COPY go/main.go ./main.go
RUN CGO_ENABLED=0 go build -o /backend .

FROM alpine:3.21
COPY --from=build /backend /backend
ENTRYPOINT ["/backend"]
