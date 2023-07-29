FROM golang:1.20

LABEL maintainer="Tobias Wiking <tobias@wiking.me>"

WORKDIR /app
COPY ./src .

RUN go mod download
RUN go build -o main .

EXPOSE 8080

CMD ["./main"]