FROM golang:1.11.4-alpine as builder

WORKDIR /go/src/github.com/Ankr-network/dccn-taskmgr
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cmd/taskmgr app-dccn-taskmgr/main.go

FROM golang:1.11.4-alpine

COPY --from=builder /go/src/github.com/Ankr-network/dccn-taskmgr/cmd/taskmgr /
CMD ["/taskmgr"]
