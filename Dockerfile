FROM golang:1.10.2-alpine3.7

WORKDIR /go/src/app
COPY . .

RUN apk update && apk add \
curl \
git

CMD /bin/bash

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN dep ensure
RUN go get -d -v
RUN go install -v
RUN go build
CMD ["app"]