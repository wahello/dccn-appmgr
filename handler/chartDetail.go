package handler

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	chartutil "k8s.io/helm/pkg/chartutil"

	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
)

// ChartDetail will return a list of specific chart versions from the chartmuseum repo
func (p *AppMgrHandler) ChartDetail(ctx context.Context,
	req *appmgr.ChartDetailRequest) (*appmgr.ChartDetailResponse, error) {

	log.Printf(">>>>>>>>>Debug into ChartDetail... %+v\nctx: %+v\n", req, ctx)

	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	rsp := &appmgr.ChartDetailResponse{}
	if req.Chart == nil || len(req.Chart.ChartName) == 0 || len(req.Chart.ChartRepo) == 0 {
		log.Printf("invalid input: null chart provided, %+v \n", req.Chart)
		return rsp, ankr_default.ErrChartNotExist
	}

	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api",
		teamId, req.Chart.ChartRepo) + "/" + req.Chart.ChartName)
	if err != nil {
		log.Printf("cannot get chart details, %s \n", err.Error())
		return rsp, ankr_default.ErrChartDetailGet
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart details response body, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadChartDetails
	}

	data := []Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart details, %s \n", err.Error())
		return rsp, ankr_default.ErrUnMarshalChartDetail
	}

	rsp.ChartName = req.Chart.ChartName
	rsp.ChartRepo = req.Chart.ChartRepo

	if len(data) == 0 {
		return rsp, nil
	}
	rsp.ChartDescription = data[0].Description

	versionDetails := make([]*common_proto.ChartVersionDetail, 0)
	for _, v := range data {
		versiondetail := common_proto.ChartVersionDetail{
			ChartVer:    v.Version,
			ChartAppVer: v.AppVersion,
		}
		versionDetails = append(versionDetails, &versiondetail)
	}
	rsp.ChartVersionDetails = versionDetails

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL, teamId,
		req.Chart.ChartRepo)+"/"+req.Chart.ChartName+"-"+req.ShowVersion+".tgz", nil)
	if err != nil {
		log.Printf("cannot create show version tarball request, %s \n", err.Error())
		return rsp, ankr_default.ErrCreateRequest
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return rsp, errors.New(ankr_default.DialError + "Cannot download chart tarball" + err.Error())
	}
	defer tarballRes.Body.Close()

	gzf, err := gzip.NewReader(tarballRes.Body)
	if err != nil {
		log.Printf("cannot open chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadDownload
	}
	defer gzf.Close()

	tarball := tar.NewReader(gzf)
	tarf := make(map[string]string)
	tarf[req.Chart.ChartName+"/README.md"] = ""
	tarf[req.Chart.ChartName+"/values.yaml"] = ""

	if err := extractFromTarfile(tarf, tarball); err != nil {
		log.Printf("cannot find readme/value in chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrNoChartReadme
	}
	if tarf[req.Chart.ChartName+"/README.md"] == "" {
		log.Printf("cannot find readme in chart tarball\n")
	}
	if tarf[req.Chart.ChartName+"/values.yaml"] == "" {
		log.Printf("cannot find value in chart tarball\n")
	}

	rsp.ReadmeMd = tarf[req.Chart.ChartName+"/README.md"]
	rsp.ValuesYaml = tarf[req.Chart.ChartName+"/values.yaml"]
	values, err := chartutil.ReadValues([]byte(rsp.ValuesYaml))
	if err != nil {
		log.Printf("parse values.yaml error: %s\n%+v\n", err, rsp.ValuesYaml)
	}
	if values != nil {
		ankrCustomValues, err := values.Table("ankrCustomValues")
		if err != nil {
			log.Printf("get ankrCustomValue from values.yaml error: %s\n%+v\n", err, ankrCustomValues)
		} else {
			if ankrCustomValues != nil {
				ankrCustomKeys := make([]*common_proto.CustomValue, 0, len(ankrCustomValues))
				for key := range ankrCustomValues {
					ankrCustomKeys = append(ankrCustomKeys, &common_proto.CustomValue{Key: key, Value: ""})
				}
				log.Printf("ankrCustomKeys: %+v\n", ankrCustomKeys)
				rsp.CustomValues = ankrCustomKeys
			}
		}
	}

	return rsp, nil
}
