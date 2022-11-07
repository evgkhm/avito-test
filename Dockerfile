FROM golang:latest

RUN go version
ENV GOPATH=/

COPY ./ ./

RUN go build -o main .

CMD ["go", "run", "cmd/main.go"]