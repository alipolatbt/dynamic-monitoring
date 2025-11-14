FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod .
COPY main.go .

RUN go mod tidy
RUN go build -o /server .

FROM alpine:3.18
RUN mkdir -p /data
COPY --from=build /server /server
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/server"]
