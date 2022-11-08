FROM golang:latest

RUN go version
ENV GOPATH=/

COPY ./ ./

RUN go mod download
RUN go build -o wallet-app ./cmd/main.go

#CMD ["go", "run", "cmd/main.go"]
CMD ["./wallet-app"]