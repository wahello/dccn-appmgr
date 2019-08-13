package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_util "github.com/Ankr-network/dccn-common/util"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"errors"
)

// DownloadChart will return a specific chart tarball from the specific chartmuseum repo
func (p *AppMgrHandler) DownloadChart(ctx context.Context,
	req *appmgr.DownloadChartRequest) (*appmgr.DownloadChartResponse, error) {

	log.Printf(">>>>>>>>>Debug into DownloadChart...%+v\nctx: %+v\n", req, ctx)

	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	rsp := &appmgr.DownloadChartResponse{}
	if len(req.ChartName) == 0 || len(req.ChartRepo) == 0 || len(req.ChartVer) == 0 {
		log.Printf("invalid input: null chart detail provided, %+v \n", req)
		return rsp, ankr_default.ErrChartDetailEmpty
	}

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL,
		teamId, req.ChartRepo)+"/"+req.ChartName+"-"+req.ChartVer+".tgz", nil)
	if err != nil {
		log.Printf("cannot create download tarball request, %s \n", err.Error())
		return rsp, ankr_default.ErrCreateRequest
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return rsp, errors.New(ankr_default.DialError + "Cannot download chart tarball" + err.Error())
	}
	defer tarballRes.Body.Close()

	chartFile, err := ioutil.ReadAll(tarballRes.Body)
	if err != nil {
		log.Printf("cannot read chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadDownload
	}

	rsp.ChartFile = chartFile

	return rsp, nil
}
