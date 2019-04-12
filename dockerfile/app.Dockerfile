FROM golang:1.11.4-alpine as builder

WORKDIR /go/src/github.com/Ankr-network/dccn-appmgr
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cmd/appmgr ./main.go

FROM golang:1.11.4-alpine

COPY --from=builder /go/src/github.com/Ankr-network/dccn-appmgr/cmd/appmgr /
COPY --from=builder /go/src/github.com/Ankr-network/dccn-appmgr /go/src/github.com/Ankr-network/dccn-appmgr
COPY --from=builder /go/src/github.com/Ankr-network/dccn-appmgr/chart_repo /go/chart_repo
CMD ["/appmgr"]
