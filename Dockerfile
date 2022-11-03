FROM golang:latest

RUN go version
ENV GOPATH=/

COPY ./ ./

RUN go build -o main .

CMD ["./main"]


#FROM golang:1.19

#RUN go version

#ENV GOPATH=/

#COPY ["src", "."]

#RUN go mod download
#RUN go build -o avito-test ./cmd/main.go

#CMD ["./avito-test"]