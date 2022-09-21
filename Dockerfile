FROM golang:1.17

RUN apt-get update -qq \
    && apt-get -y install libopus-dev libopusfile-dev

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go mod tidy
RUN go build -o main ./cmd
CMD ["/app/main"]