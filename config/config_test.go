package config

import (
	"testing"
)

var (
	path = "config.json"
)

func TestNew(t *testing.T) {
	conf, err := New(path)
	if err != nil {
		t.Error(err.Error())
	}

	t.Logf("Load config %+v", conf)
}

