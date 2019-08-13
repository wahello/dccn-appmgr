package handler

import (
	"encoding/json"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"context"
)

// ChartList will return a list of charts from the specific chartmuseum repo
func (p *AppMgrHandler) ChartList(ctx context.Context, req *appmgr.ChartListRequest) (*appmgr.ChartListResponse, error) {

	log.Printf(">>>>>>>>>Debug into ChartList...%+v\nctx: %+v\n", req, ctx)

	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	rsp := &appmgr.ChartListResponse{}

	if len(req.ChartRepo) == 0 {
		req.ChartRepo = "stable"
	}
	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api", teamId, req.ChartRepo))
	if err != nil {
		log.Printf("cannot get chart list, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotGetChartList
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart list response body, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadChartList
	}

	data := map[string][]Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart list, %s \n", err.Error())
		return rsp, ankr_default.ErrUnMarshalChartList
	}

	charts := make([]*common_proto.Chart, 0)

	for _, v := range data {
		chart := common_proto.Chart{
			ChartName:             v[0].Name,
			ChartRepo:             req.ChartRepo,
			ChartDescription:      v[0].Description,
			ChartIconUrl:          v[0].Icon,
			ChartLatestVersion:    v[0].Version,
			ChartLatestAppVersion: v[0].AppVersion,
		}
		charts = append(charts, &chart)
	}
	sort.Sort(chartList(charts))
	rsp.Charts = charts

	return rsp, nil
}
