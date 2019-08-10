package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	"errors"
	)

func checkNsId(userId, nsId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if nsId == "" || nsId != userId {
		return errors.New("User does not own this namespace")
	}

	return nil
}