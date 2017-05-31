// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/google/syzkaller/dashboard"
	"github.com/google/syzkaller/pkg/config"
	"github.com/google/syzkaller/pkg/gcs"
	"github.com/google/syzkaller/pkg/git"
	"github.com/google/syzkaller/pkg/hash"
	"github.com/google/syzkaller/pkg/kernel"
	. "github.com/google/syzkaller/pkg/log"
)

var (
	flagConfig = flag.String("config", "", "config file")
)

const (
	pollDelay    = 1 * time.Hour
	rebuildAfter = 24 * time.Hour
)

type Config struct {
	Dashboard_Addr string
	Dashboard_Key  string
	Compiler       string
	Userspace      string
	GCS_Bucket     string
}

type Context struct {
	cfg  *Config
	dash *dashboard.Dashboard
	GCS  *gcs.Client
}

func main() {
	flag.Parse()
	EnableLogCaching(1000, 1<<20)
	cfg := new(Config)
	if err := config.Load(*flagConfig, cfg); err != nil {
		Fatalf("%v", err)
	}
	dash := &dashboard.Dashboard{
		Addr:   cfg.Dashboard_Addr,
		Client: "builder",
		Key:    cfg.Dashboard_Key,
	}
	GCS, err := gcs.NewClient()
	if err != nil {
		Fatalf("failed to create GCS client: %v", err)
	}
	ctx := &Context{
		cfg:  cfg,
		dash: dash,
		GCS:  GCS,
	}
	for ; ; time.Sleep(pollDelay) {
		builds, err := dash.BuilderPoll()
		if err != nil {
			Logf(0, "failed to poll dashboard: %v", err)
			continue
		}
		for _, build := range builds {
			handleBuild(ctx, build)
		}
	}
}

func handleBuild(ctx *Context, build dashboard.Build) {
	Logf(0, "%v: %v at %v", build.ID, build.Commit, build.Time)
	if time.Since(build.Time) < rebuildAfter {
		Logf(0, "fresh")
		return
	}
	dir := build.ID
	commit, err := git.Poll(dir, build.Repo, build.Branch)
	if err != nil {
		Logf(0, "failed to poll: %v", err)
		return
	}
	if commit == build.Commit {
		Logf(0, "no changes")
		return
	}
	Logf(0, "building on %v...", commit)
	compilerID, err := kernel.CompilerIdentity(ctx.cfg.Compiler)
	if err != nil {
		Logf(0, "failed to get compiler identity: %v", err)
		return
	}
	if err := kernel.Build(dir, ctx.cfg.Compiler, build.Config); err != nil {
		Logf(0, "build failed: %v", err)
		return
	}
	config, err := ioutil.ReadFile(filepath.Join(dir, ".config"))
	if err != nil {
		Logf(0, "failed to read .config: %v", err)
		return
	}
	Logf(0, "building image...")
	tag := build.ID + "/" + commit
	imagePath := filepath.Join(dir, "image.tar.gz")
	if err := kernel.CreateImage(dir, ctx.cfg.Userspace, tag, imagePath); err != nil {
		Logf(0, "image build failed: %v", err)
		return
	}
	imageID := hash.String([]byte(build.ID + commit))
	gcsFilename := filepath.Join(ctx.cfg.GCS_Bucket, imageID)
	if err := ctx.GCS.UploadFile(imagePath, gcsFilename); err != nil {
		Logf(0, "failed to upload image to '%v': %v", gcsFilename, err)
		return
	}
	image := &dashboard.Image{
		ID:       imageID,
		BuildID:  build.ID,
		File:     gcsFilename,
		Compiler: compilerID,
		Commit:   commit,
		Config:   string(config),
	}
	if err := ctx.dash.UploadImage(image); err != nil {
		Logf(0, "failed to upload image to dashboard: %v", err)
		return
	}
}
