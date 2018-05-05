FROM golang:1.10.2-alpine3.7
ADD . /src
WORKDIR /src
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN go build
CMD ["bin/simmetrica"]