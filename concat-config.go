package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const baseFile = "base.toml"

func watchExec(file string, function func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	delay := 50 * time.Millisecond
	timer := time.NewTimer(delay)
	timer.Stop()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					if !timer.Stop() {
						function()
					}
					timer.Reset(delay)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error: ", err)
			}
		}
	}()

	err = watcher.Add(file)
	if err != nil {
		log.Fatal(err)
	}

	<-make(chan struct{})
}

func concatConfig(files []string) {
	quotedFiles := make([]string, len(files))
	for i, file := range files {
		quotedFiles[i] = `"` + file + `"`
	}

	base, err := os.ReadFile(baseFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		transient, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		baseStr := string(base)
		transientStr := string(transient)

		beginingMarker := "# Begining of " + baseFile
		endMarker := "# End of " + baseFile

		if strings.Contains(transientStr, beginingMarker) && strings.Contains(transientStr, endMarker) {
			startIndex := strings.Index(transientStr, beginingMarker) + len(beginingMarker)
			endIndex := strings.Index(transientStr, endMarker)
			transientStr = transientStr[:startIndex] + "\n" + baseStr + "\n" + transientStr[endIndex:]
		} else {
			transientStr += "\n" + beginingMarker + "\n" + baseStr + "\n" + endMarker
		}

		err = os.WriteFile(file, []byte(transientStr), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Concatenated \""+baseFile+"\" to the files", strings.Join(quotedFiles, ", "))
}

func main() {
	files := []string{"transient.toml", "main.toml"}

	var argLength int = len(os.Args)
	if argLength > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			log.Println("Usage: concat-config [option] <files>")
			os.Exit(0)
		} else if os.Args[1] == "-w" || os.Args[1] == "--watch" {
			if argLength > 2 {
				files = os.Args[2:]
			}
			concatConfig(files)
			fmt.Println("Watching \"" + baseFile + "\"...")
			watchExec(baseFile, func() {
				concatConfig(files)
			})
		} else {
			files = os.Args[1:]
			concatConfig(files)
		}
	} else {
		concatConfig(files)
	}
}
