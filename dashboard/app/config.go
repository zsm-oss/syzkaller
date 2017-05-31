// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// +build appengine

package dash

import (
	"fmt"
	"encoding/json"

	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type Config struct {
	Pools []VMPool `json:",omitempty"`
}

type VMPool struct {
	Name string
	Key string
}

func parseConfig(data string) (*Config, error) {
	cfg := new(Config)
	if err := json.Unmarshal([]byte(data), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}
	return cfg, nil
}

func (cfg *Config) Serialize() string {
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(data)
}

type ConfigEntity struct {
	Text string
}

func storeConfig(c context.Context, cfg *Config) error {
	ent := &ConfigEntity{cfg.Serialize()}
	if _, err := datastore.Put(c, datastore.NewKey(c, "Config", "", 1, nil), ent); err != nil {
		return err
	}
	return nil	
}

func loadConfig(c context.Context) (*Config, error) {
	ent := &ConfigEntity{}
	err := datastore.Get(c, datastore.NewKey(c, "Config", "", 1, nil), ent)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}
	if ent.Text == "" {
		return &Config{}, nil
	}
	cfg, err := parseConfig(ent.Text)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
