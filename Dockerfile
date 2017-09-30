FROM golang:1.8
RUN mkdir -p /go/src/github.com/productiongo/bestbooktolearn
WORKDIR /go/src/github.com/productiongo/bestbooktolearn
COPY . .
RUN go-wrapper install
CMD ["go-wrapper", "run"]