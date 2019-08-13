package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	"errors"
	)

func checkNsId(teamId, nsId string) error {
	if teamId == "" {
		return ankr_default.ErrUserNotExist
	}

	if nsId == "" || nsId != teamId {
		return errors.New("User does not own this namespace")
	}

	return nil
}