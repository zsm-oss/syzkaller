// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/google/syzkaller/pkg/config"
	mgrconfig "github.com/google/syzkaller/syz-manager/config"
)

type Config struct {
	Name           string
	Http_Port      int
	Dashboard_Addr string
	Dashboard_Key  string
	Hub_Addr       string
	Hub_Key        string
	GCS_Bucket     string
	Managers       []ManagerConfig
}

type ManagerConfig struct {
	Name           string
	Repo           string
	Branch         string
	Compiler       string
	Userspace      string
	Kernel_Config  string
	Manager_Config mgrconfig.Config
}

func loadConfig(filename string) (*Config, error) {
	cfg := new(Config)
	if err := config.LoadFile(filename, cfg); err != nil {
		return nil, err
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("param 'name' is empty")
	}
	if cfg.Http_Port == 0 {
		return nil, fmt.Errorf("param 'http_port' is empty")
	}
	if len(cfg.Managers) == 0 {
		return nil, fmt.Errorf("no managers specified")
	}
	for i, mgr := range cfg.Managers {
		if mgr.Name == "" {
			return nil, fmt.Errorf("param 'managers[%v].name' is empty", i)
		}
		if mgr.Manager_Config.Type == "" {
			return nil, fmt.Errorf("param 'managers[%v].manager_config.type' is empty", i)
		}
	}
	return cfg, nil
}
