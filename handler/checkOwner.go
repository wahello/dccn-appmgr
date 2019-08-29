package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"log"
)

func (p *AppMgrHandler) checkOwner(teamId, appId string) (*common_proto.AppReport, error) {
	appRecord, err := p.db.GetApp(appId)

	if err != nil {
		return nil, err
	}

	log.Printf("appid : %s user id -%s-   user_token_id -%s-  ", appId, appRecord.TeamID, teamId)

	if appRecord.TeamID != teamId {
		return nil, ankr_default.ErrUserNotOwn
	}

	appMessage := convertToAppMessage(appRecord, p.db)

	return &appMessage, nil
}
