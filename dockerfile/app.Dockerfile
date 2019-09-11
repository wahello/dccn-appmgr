FROM golang:1.12-alpine as builder
ARG CHARTMUSEUM_URL
WORKDIR /go/src/github.com/Ankr-network/dccn-appmgr
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s -X github.com/Ankr-network/dccn-appmgr/handler.chartmuseumURL=${CHARTMUSEUM_URL}" -o cmd/appmgr ./main.go

FROM golang:1.12-alpine

COPY --from=builder /go/src/github.com/Ankr-network/dccn-appmgr/cmd/appmgr /
COPY --from=builder /go/src/github.com/Ankr-network/dccn-appmgr /go/src/github.com/Ankr-network/dccn-appmgr
CMD ["/appmgr"]
