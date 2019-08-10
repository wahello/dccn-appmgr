package handler

import (
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	micro2 "github.com/Ankr-network/dccn-common/ankr-micro"
)

var chartmuseumURL string

type AppMgrHandler struct {
	db        db.DBService
	deployApp *micro2.Publisher
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

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

func New(db db.DBService, deployApp *micro2.Publisher) *AppMgrHandler {
	return &AppMgrHandler{
		db:        db,
		deployApp: deployApp,
	}
}

/*
type chartList []*common_proto.Chart

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
*/



