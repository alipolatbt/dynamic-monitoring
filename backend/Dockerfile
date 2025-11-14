FROM golang:1.25-alpine

WORKDIR /app

# copy go.mod first to leverage layer caching
COPY go.mod .
COPY . .
# generate go.sum and download modules inside the image
RUN go mod tidy && go build -o main .

EXPOSE 8080
CMD ["./main"]
FROM golang:1.25-alpine

WORKDIR /app

# copy go.mod first to leverage layer caching
COPY go.mod .
COPY . .
# generate go.sum and download modules inside the image
RUN go mod tidy && go build -o main .

EXPOSE 8080
CMD ["./main"]