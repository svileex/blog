FROM golang:alpine
RUN mkdir /blog
ADD . /blog
WORKDIR /blog/cmd/microblog
RUN go mod download
RUN apk add --no-cache build-base
RUN go build
RUN go test -v
CMD [ "./microblog" ]