package handler

import ankr_default "github.com/Ankr-network/dccn-common/protos"

func checkId(teamId, appId string) error {
	if teamId == "" {
		return ankr_default.ErrUserNotExist
	}

	if appId == "" {
		return ankr_default.ErrUserNotOwn
	}

	return nil
}

