FROM golang:1.10.2-alpine3.7

WORKDIR /go/src/app
COPY . .

ENV GOBIN $GOPATH/bin

RUN apk update && apk add \
curl \
git

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN dep ensure
RUN go install -v cmd/controller/simmetrica.go

ENTRYPOINT $GOBIN/simmetrica