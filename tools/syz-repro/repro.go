// Copyright 2015 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/syzkaller/pkg/csource"
	"github.com/google/syzkaller/pkg/log"
	"github.com/google/syzkaller/pkg/mgrconfig"
	"github.com/google/syzkaller/pkg/osutil"
	"github.com/google/syzkaller/pkg/report"
	"github.com/google/syzkaller/pkg/repro"
	"github.com/google/syzkaller/vm"
)

var (
	flagConfig = flag.String("config", "", "manager configuration file (manager.cfg)")
	flagCount  = flag.Int("count", 0, "number of VMs to use (overrides config count param)")
	flagDebug  = flag.Bool("debug", false, "print debug output")
	flagTitle  = flag.Bool("title", false, "turn on manual checking for crash title")
)

var seenCrashTitles = make(map[string]bool)

func confirm(crashTitle string) (bool, error) {
	for {
		prompt := fmt.Sprintf("program crashed with new title %v; keep crashes with this title?", crashTitle)
		fmt.Printf("%s [yes/no]: ", prompt)
		var resp string
		if _, err := fmt.Scanln(&resp); err != nil {
			return false, fmt.Errorf("could not read user response")
		}
		if resp == "yes" {
			return true, nil
		} else if resp == "no" {
			return false, nil
		}
	}
}

func checkTitle(crashTitle string) (bool, error) {
	if resp, ok := seenCrashTitles[crashTitle]; ok {
		return resp, nil
	}

	wantTitle, err := confirm(crashTitle)
	if err != nil {
		return false, err
	}
	seenCrashTitles[crashTitle] = wantTitle
	return wantTitle, nil
}

func main() {
	os.Args = append(append([]string{}, os.Args[0], "-vv=10"), os.Args[1:]...)
	flag.Parse()
	if len(flag.Args()) != 1 || *flagConfig == "" {
		log.Fatalf("usage: syz-repro -config=manager.cfg execution.log")
	}
	cfg, err := mgrconfig.LoadFile(*flagConfig)
	if err != nil {
		log.Fatalf("%v: %v", *flagConfig, err)
	}
	logFile := flag.Args()[0]
	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		log.Fatalf("failed to open log file %v: %v", logFile, err)
	}
	vmPool, err := vm.Create(cfg, *flagDebug)
	if err != nil {
		log.Fatalf("%v", err)
	}
	vmCount := vmPool.Count()
	if *flagCount > 0 && *flagCount < vmCount {
		vmCount = *flagCount
	}
	if vmCount > 4 {
		vmCount = 4
	}
	vmIndexes := make([]int, vmCount)
	for i := range vmIndexes {
		vmIndexes[i] = i
	}
	reporter, err := report.NewReporter(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	osutil.HandleInterrupts(vm.Shutdown)

	var checkTitleFn func(string) (bool, error)

	if *flagTitle {
		checkTitleFn = checkTitle
	}

	res, stats, err := repro.Run(data, cfg, nil, reporter, vmPool, vmIndexes, checkTitleFn)
	if err != nil {
		log.Logf(0, "reproduction failed: %v", err)
	}
	if stats != nil {
		fmt.Printf("extracting prog: %v\n", stats.ExtractProgTime)
		fmt.Printf("minimizing prog: %v\n", stats.MinimizeProgTime)
		fmt.Printf("simplifying prog options: %v\n", stats.SimplifyProgTime)
		fmt.Printf("extracting C: %v\n", stats.ExtractCTime)
		fmt.Printf("simplifying C: %v\n", stats.SimplifyCTime)
	}
	if res == nil {
		return
	}

	fmt.Printf("opts: %+v crepro: %v\n\n", res.Opts, res.CRepro)
	fmt.Printf("%s\n", res.Prog.Serialize())
	if res.CRepro {
		src, err := csource.Write(res.Prog, res.Opts)
		if err != nil {
			log.Fatalf("failed to generate C repro: %v", err)
		}
		if formatted, err := csource.Format(src); err == nil {
			src = formatted
		}
		fmt.Printf("%s\n", src)
	}
}
