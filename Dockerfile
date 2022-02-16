FROM golang:1.14

RUN mkdir ./stacksrv
COPY . ./stacksrv

ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

WORKDIR /go/stacksrv

RUN cd ./cmd/servd && go build -race -o stacksrv servd.go
RUN chmod 0766 ./cmd/servd/stacksrv

CMD [ "./cmd/servd/stacksrv", "-service=:8080", "-control=:8081" ]