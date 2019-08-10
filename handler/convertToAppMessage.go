package handler

import (
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"log"
	"strings"
)


func convertToAppMessage(app db.AppRecord, pdb db.DBService) common_proto.AppReport {
	message := common_proto.AppDeployment{}
	message.AppId = app.ID
	message.AppName = app.Name
	message.Attributes = &common_proto.AppAttributes{
		CreationDate:     app.CreationDate,
		LastModifiedDate: app.LastModifiedDate,
	}
	message.Uid = app.UID
	message.ChartDetail = &app.ChartDetail
	namespaceRecord, err := pdb.GetNamespace(app.NamespaceID)
	if err != nil {
		log.Printf("get namespace record failed, %s", err.Error())
	}
	namespaceReport := convertFromNamespaceRecord(namespaceRecord)
	message.Namespace = namespaceReport.Namespace
	appReport := common_proto.AppReport{
		AppDeployment: &message,
		AppStatus:     app.Status,
		AppEvent:      app.Event,
		Detail:        app.Detail,
		Report:        app.Report,
	}
	if len(app.Detail) > 0 && len(message.Namespace.ClusterId) > 0 &&
		strings.Contains(app.Detail, app.ID+"."+message.Namespace.ClusterId+".ankr.com") {
		appReport.Endpoint = app.ID + "." + message.Namespace.ClusterId + ".ankr.com"
	}

	return appReport
}
