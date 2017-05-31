// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// +build appengine

package dash

import (
	"net/http"

	"golang.org/x/net/context"
)

func init() {
	http.Handle("/", handlerWrapper(handleMain))
}

type uiMain struct {
	Header *uiHeader
	Config string
}

func handleMain(c context.Context, w http.ResponseWriter, r *http.Request) error {
	h, err := commonHeader(c)
	if err != nil {
		return nil
	}
	data := &uiMain{
		Header: h,
	}
	switch r.FormValue("action") {
	case "update_config":
		if err := updateConfig(c, r); err != nil {
			return err
		}
	}
	cfg, err := loadConfig(c)
	if err != nil {
		return err
	}
	data.Config = cfg.Serialize()
	return templates.ExecuteTemplate(w, "main.html", data)
}

func updateConfig(c context.Context, r *http.Request) error {
	cfg, err := parseConfig(r.FormValue("config"))
	if err != nil {
		return err
	}
	if err := storeConfig(c, cfg); err != nil {
		return err
	}
	return nil
}
