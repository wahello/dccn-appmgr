package handler

import (
	"errors"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
)

func checkNsId(teamId, nsTeamID string) error {
	if teamId == "" {
		return ankr_default.ErrUserNotExist
	}

	if nsTeamID == "" || nsTeamID != teamId {
		return errors.New(ankr_default.ArgumentError + "User does not own this namespace")
	}

	return nil
}
