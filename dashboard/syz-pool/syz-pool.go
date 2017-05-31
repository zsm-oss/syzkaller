// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"flag"
	//"io/ioutil"
	//"path/filepath"
	//"time"

	"github.com/google/syzkaller/dashboard"
	//"github.com/google/syzkaller/pkg/gcs"
	//"github.com/google/syzkaller/pkg/git"
	//"github.com/google/syzkaller/pkg/hash"
	//"github.com/google/syzkaller/pkg/kernel"
	. "github.com/google/syzkaller/pkg/log"
)

var (
	flagConfig = flag.String("config", "", "config file")
)

func main() {
	flag.Parse()
	EnableLogCaching(1000, 1<<20)
	cfg, err := loadConfig(*flagConfig)
	if err != nil {
		Fatalf("failed to load config: %v", err)
	}
	dash, err := dashboard.New(cfg.Name, cfg.Dashboard_Addr, cfg.Dashboard_Key)
	if err != nil {
		Fatalf("failed to create dashboard client: %v", err)
	}
	_ = dash
	//var GCE *gce.Context
	//var GCS *gcs.Client
	for _, mgrCfg := range cfg.Managers {
		_ = mgrCfg
		//if mgrCfg.
	}
}
