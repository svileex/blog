FROM golang:alpine
RUN mkdir /blog
ADD . /blog
WORKDIR /blog/cmd/microblog
RUN go build
CMD [ "./microblog" ]