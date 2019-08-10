package handler

import ankr_default "github.com/Ankr-network/dccn-common/protos"

func checkId(userId, appId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if appId == "" {
		return ankr_default.ErrUserNotOwn
	}

	return nil
}

