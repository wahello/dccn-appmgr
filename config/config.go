package config

import (
	"os"
	"strconv"

	dbcommon "github.com/Ankr-network/dccn-common/db"
)

type Config struct {
	DB dbcommon.Config
}

var Default = Config{
	DB: dbcommon.Config{
		Host:       "127.0.0.1:27017",
		DB:         "dccn",
		Collection: "task",
		Timeout:    5,
		PoolLimit:  4096,
	},
}

func Load() (Config, error) {
	if host := os.Getenv("DB_HOST"); len(host) != 0 {
		Default.DB.Host = host
	}
	if dbName := os.Getenv("DB_NAME"); len(dbName) != 0 {
		Default.DB.DB = dbName
	}
	if collection := os.Getenv("DB_COLLECTION"); len(collection) != 0 {
		Default.DB.Collection = collection
	}
	if timeout := os.Getenv("DB_TIMEOUT"); len(timeout) != 0 {
		if t, err := strconv.Atoi(timeout); err == nil {
			Default.DB.Timeout = t
		}
	}
	if poolLimit := os.Getenv("DB_POOL_LIMIT"); len(poolLimit) != 0 {
		if t, err := strconv.Atoi(poolLimit); err != nil {
			return Default, err
		} else {
			Default.DB.PoolLimit = t
		}
	}

	return Default, nil
}
