FROM golang:1.20.6-alpine

WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY main.go /app/

RUN go build -o redis-migrate

ENTRYPOINT [ "./redis-migrate" ]
