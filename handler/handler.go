package handler

import (
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/Ankr-network/dccn-common/broker"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"sort"
)

type AppMgrHandler struct {
	db        db.DBService
	deployApp broker.Publisher
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

func New(db db.DBService, deployApp broker.Publisher) *AppMgrHandler {
	return &AppMgrHandler{
		db:        db,
		deployApp: deployApp,
	}
}

var chartmuseumURL string

type chartList []*common_proto.Chart

// Maintainer is a struct representing a maintainer inside a chart
type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Chart is a struct representing a chartmuseum chart in the manifest
type Chart struct {
	Name        string       `json:"name"`
	Home        string       `json:"home"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Keywords    []string     `json:"keywords"`
	Maintainers []Maintainer `json:"maintainers"`
	Icon        string       `json:"icon"`
	AppVersion  string       `json:"appVersion"`
	URLS        []string     `json:"urls"`
	Created     string       `json:"created"`
	Digest      string       `json:"digest"`
}

func (c chartList) Len() int {
	return len(c)
}

func (c chartList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c chartList) Less(i, j int) bool {
	data := []string{c[i].ChartName, c[j].ChartName}
	return sort.StringsAreSorted(data)
}
