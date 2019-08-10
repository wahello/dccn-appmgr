package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"context"
	"log"
	"net/http"
	"errors"
)

// DeleteChart delete a specific chart version from the specific chartmuseum repo
func (p *AppMgrHandler) DeleteChart(ctx context.Context,
	req *appmgr.DeleteChartRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into DeleteChart...%+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)

	query, err := http.Get(getChartURL(chartmuseumURL+"/api", uid,
		req.ChartRepo) + "/" + req.ChartName + "/" + req.ChartVer)
	if query.StatusCode != 200 {
		log.Printf("chart not exist, delete failed.\n")
		return &common_proto.Empty{}, ankr_default.ErrChartNotExist
	}

	delReq, err := http.NewRequest("DELETE", getChartURL(chartmuseumURL+"/api",
		uid, req.ChartRepo)+"/"+req.ChartName+"/"+req.ChartVer, nil)
	if err != nil {
		log.Printf("cannot create delete chart request, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCreateRequest
	}

	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		log.Printf("cannot delete chart file, %s \n", err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.LogicError + "Cannot delete chart file" + err.Error())
	}
	defer delRes.Body.Close()

	return &common_proto.Empty{}, nil
}
