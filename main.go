//go:build windows
// +build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	winlog "github.com/ofcoursedude/gowinlog"
)

// truncate event log message if longer than max_msg_len characters
const max_msg_len = 2000

var optEvtChan string
var optEvtProv string
var optTime int

func Usage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf("tail windows event log\n")
	fmt.Printf("Usage: %s -n <name> -p <provider> -t <time (optional)>\n\n", exe)
	flag.PrintDefaults()
}

var reNameBlacklist = regexp.MustCompile(`(&|>|<|\/|:|\n|\r)*`)

func SanitizeName(name string, limit int) string {
	name = reNameBlacklist.ReplaceAllString(name, "")
	result := name
	chars := 0
	for i := range name {
		if chars >= limit {
			result = name[:i]
			break
		}
		chars++
	}
	return result
}

func main() {
	flag.Usage = Usage

	flag.StringVar(&optEvtProv, "p", "", `name of the windows event log provider (i.e. Application name").`)
	flag.StringVar(&optEvtChan, "n", "Application", `name of the windows event log channel (i.e. System, Application, ... - see powershell "get-winevent -listlog *").`)
	flag.IntVar(&optTime, "t", 1440, "display recent events from last N minutes (defaults to 24 hours)")
	flag.Parse()

	fmt.Fprintln(os.Stderr, "starting...")
	watcher, err := winlog.NewWinLogWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR - Couldn't create watcher: %v\n", err)
		os.Exit(1)
	}
	err = watcher.SubscribeFromBeginning(optEvtChan, fmt.Sprintf("*[System/TimeCreated[timediff(@SystemTime) < %d]]", (optTime*60*1000)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR - Couldn't subscribe to Application: %v", err)
		os.Exit(1)
	}
	for {
		select {
		case evt := <-watcher.Event():
			if optEvtChan != "Application" || optEvtProv == evt.ProviderName {
				fmt.Println(evt.Created, evt.ComputerName, evt.Channel, evt.EventId, evt.Opcode, evt.LevelText, evt.ProviderName, SanitizeName(strings.TrimSpace(evt.Msg), max_msg_len))
			}
		case err := <-watcher.Error():
			fmt.Printf("Error: %v\n\n", err)
		case <-time.After(1 * time.Second):
			continue

		}
	}
}
