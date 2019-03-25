package dbcommon

import (
	"os"
	"strconv"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"
)

var once sync.Once

var (
	DEFAULT_DB           = "dccn"
	DEFAULTT_COLLECTIOIN = "user"
	DEFAULT_HOST         = "localhost:27017"
	DEFAULT_POOL_LIMIT   = 4096
	DEFAULT_TIMEOUT      = 5
)

// Config uses to init a db connect
type Config struct {
	// DB db name
	DB string `json:"db_name"`
	// Collection db table
	Collection string `json:"collection_name"`
	// Host holds the addresses for the seed servers.
	Host string `json:"host"`
	// PoolLimit defines the per-server socket pool limit. Defaults to 4096.
	PoolLimit int `json:"pool_limit"`
	// Timeout is the amount of time to wait for a server to respond
	Timeout int `json:"timeout"`
}

// CreateDBConnection returns a db connection, it is recommended to use for once, and then use copy or clone to reuse it
// remembers to close after every copy() or clone()
func CreateDBConnection(conf Config) (s *mgo.Session, err error) {
	once.Do(func() {
		info := mgo.DialInfo{
			Addrs:     []string{conf.Host},
			Timeout:   time.Duration(conf.Timeout) * time.Second,
			PoolLimit: conf.PoolLimit,
		}

		if s, err = mgo.DialWithInfo(&info); err != nil {
			return
		}
		s.SetMode(mgo.Monotonic, true)
		s.SetSafe(&mgo.Safe{})
	})
	return
}

// LoadFromEnv Load DB Config from env.
func LoadFromEnv() (Config, error) {
	var conf Config

	if host := os.Getenv("DB_HOST"); len(host) != 0 {
		conf.Host = host
	}
	if dbName := os.Getenv("DB_NAME"); len(dbName) != 0 {
		conf.DB = dbName
	}
	if collection := os.Getenv("DB_COLLECTION"); len(collection) != 0 {
		conf.Collection = collection
	}

	if timeout := os.Getenv("DB_TIMEOUT"); len(timeout) != 0 {
		if t, err := strconv.Atoi(timeout); err != nil {
			return conf, err
		} else {
			conf.Timeout = t
		}
	}
	if poolLimit := os.Getenv("DB_POOL_LIMIT"); len(poolLimit) != 0 {
		if t, err := strconv.Atoi(poolLimit); err != nil {
			return conf, err
		} else {
			conf.PoolLimit = t
		}
	}
	return conf, nil
}
