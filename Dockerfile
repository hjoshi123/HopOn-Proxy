FROM golang:1.11

LABEL author="Hemant Joshi <hemant@nuvonic.net>"

RUN mkdir -p /go/src/github.com/hjoshi123/HopOn-Proxy
WORKDIR /go/src/github.com/hjoshi123/HopOn-Proxy

COPY ./build build
COPY *.* ./

RUN go install

EXPOSE 80 443

CMD [ "go run main.go forward.go" ]