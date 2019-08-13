package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"gopkg.in/mgo.v2/bson"
)

func (p *AppMgrHandler) UpdateApp(ctx context.Context,
	req *appmgr.UpdateAppRequest) (*common_proto.Empty, error) {
	log.Printf(">>>>>>>>>Debug into UpdateApp: %+v\nctx: %+v\n", req, ctx)
	_, teamId := common_util.GetUserIDAndTeamID(ctx)

	if req.AppDeployment == nil || (req.AppDeployment.ChartDetail == nil ||
		len(req.AppDeployment.ChartDetail.ChartVer) == 0) && len(req.AppDeployment.AppName) == 0 {
		log.Printf("invalid input: no valid update app parameters, %+v \n", req.AppDeployment)
		return &common_proto.Empty{}, errors.New("invalid input: no valid update app parameters")
	}

	if err := checkId(teamId, req.AppDeployment.AppId); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	appReport, err := p.checkOwner(teamId, req.AppDeployment.AppId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if appReport.AppStatus != common_proto.AppStatus_APP_RUNNING &&
		appReport.AppStatus != common_proto.AppStatus_APP_UPDATE_FAILED {
		log.Println("app status is not running, cannot update")
		return &common_proto.Empty{}, ankr_default.ErrStatusNotSupportOperation
	}

	appDeployment := appReport.AppDeployment

	clusterConnection, err := p.db.GetClusterConnection(appDeployment.Namespace.ClusterId)
	if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
		log.Println("cluster connection not available, app can not be updated")
		return &common_proto.Empty{}, errors.New("cluster connection not available, app can not be updated")
	}

	if len(req.AppDeployment.AppName) > 0 {
		if err := p.db.Update("app", appDeployment.AppId,
			bson.M{"$set": bson.M{"name": req.AppDeployment.AppName}}); err != nil {
			log.Printf(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	if req.AppDeployment.ChartDetail != nil && len(req.AppDeployment.ChartDetail.ChartVer) > 0 &&
		req.AppDeployment.ChartDetail.ChartVer != appDeployment.ChartDetail.ChartVer {
		appChart, err := http.Get(getChartURL(chartmuseumURL, teamId,
			appDeployment.ChartDetail.ChartRepo) + "/" + appDeployment.ChartDetail.ChartName + "-" + req.AppDeployment.ChartDetail.ChartVer + ".tgz")
		if err != nil {
			log.Printf("cannot get app chart %s from chartmuseum\nerror: %s\n", req.AppDeployment.ChartDetail.ChartName, err.Error())
			return &common_proto.Empty{}, ankr_default.ErrChartMuseumGet
		}
		if appChart.StatusCode != 200 {
			log.Printf("invalid input: app chart not exist \n")
			return &common_proto.Empty{}, ankr_default.ErrChartNotExist
		}

		appDeployment.ChartDetail.ChartVer = req.AppDeployment.ChartDetail.ChartVer
		if appDeployment.CustomValues != nil {
			var customValues []*common_proto.CustomValue
			for _, customValue := range appDeployment.CustomValues {
				customValues = append(customValues, &common_proto.CustomValue{Key: "ankrCustomValues." + customValue.Key, Value: customValue.Value})
			}
		}
		event := common_proto.DCStream{
			OpType:    common_proto.DCOperation_APP_UPDATE,
			OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
		}

		if err := p.deployApp.Publish(&event); err != nil {
			log.Println(err.Error())
			return &common_proto.Empty{}, ankr_default.ErrPublish
		}

		// TODO: wait deamon notify
		if err := p.db.UpdateApp(appDeployment); err != nil {
			log.Println(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	return &common_proto.Empty{}, nil
}
