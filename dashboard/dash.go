// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// package dashboard defines data structures used in dashboard communication
// and provides client interface.
package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Dashboard struct {
	Client string
	Addr   string
	Key    string
}

type Crash struct {
	Tag    string
	Desc   string
	Log    []byte
	Report []byte
}

type Repro struct {
	Crash      Crash
	Reproduced bool
	Opts       string
	Prog       []byte
	CProg      []byte
}

type Patch struct {
	Title string
	Diff  []byte
}

func New(client, addr, key string) (*Dashboard, error) {
	dash := &Dashboard{
		Client: client,
		Addr:   addr,
		Key:    key,
	}
	return dash, nil
}

func (dash *Dashboard) ReportCrash(crash *Crash) error {
	return dash.query("add_crash", crash, nil)
}

func (dash *Dashboard) ReportRepro(repro *Repro) error {
	return dash.query("add_repro", repro, nil)
}

func (dash *Dashboard) PollPatches() (string, error) {
	hash := ""
	err := dash.query("poll_patches", nil, &hash)
	return hash, err
}

func (dash *Dashboard) GetPatches() ([]Patch, error) {
	var patches []Patch
	err := dash.query("get_patches", nil, &patches)
	return patches, err
}

func (dash *Dashboard) query(method string, req, reply interface{}) error {
	values := make(url.Values)
	values.Add("client", dash.Client)
	values.Add("key", dash.Key)
	values.Add("method", method)
	var body io.Reader
	if req != nil {
		data, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}
		body = bytes.NewReader(data)
	}
	resp, err := http.Post(fmt.Sprintf("%v/api?%v", dash.Addr, values.Encode()), "application/json", body)
	if err != nil {
		return fmt.Errorf("http request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed with %v: %s", resp.Status, data)
	}
	if reply != nil {
		if err := json.NewDecoder(resp.Body).Decode(reply); err != nil {
			return fmt.Errorf("failed to unmarshal response: %v", err)
		}
	}
	return nil
}

// New dashboad.

type Build struct {
	ID     string
	Repo   string
	Branch string
	Commit string
	Config string
	Time   time.Time
}

type Image struct {
	ID       string // unqiue hash
	BuildID  string // copied from Build.ID
	File     string // GCS path
	Compiler string // compiler identification
	Commit   string // commit hash on which it was built
	Config   string // actual config used to build
}

func (dash *Dashboard) BuilderPoll() ([]Build, error) {
	var builds []Build
	err := dash.query("builder_poll", nil, &builds)
	return builds, err
}

func (dash *Dashboard) UploadImage(image *Image) error {
	return dash.query("upload_image", image, nil)
}
