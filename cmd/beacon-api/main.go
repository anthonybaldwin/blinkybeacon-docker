package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

const fsbeaconBin = "/usr/local/bin/fsbeacon"

// An indefinite run (spin/strobe with no duration) keeps a CLI process
// alive to refresh the beacon's keepalive; track it so /stop can end it.
var (
	procMu   sync.Mutex
	indefCmd *exec.Cmd
)

// stopIndefinite terminates any running indefinite beacon process and
// waits for it to exit, so its keepalive can't override a stop.
func stopIndefinite() {
	procMu.Lock()
	defer procMu.Unlock()
	if indefCmd != nil {
		indefCmd.Process.Signal(syscall.SIGTERM)
		indefCmd.Wait()
		indefCmd = nil
	}
}

func beaconHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Usage: /[command]/[args]", http.StatusBadRequest)
		return
	}

	// Whitelist commands
	validCommands := map[string]bool{
		"strobe": true,
		"spin":   true,
		"stop":   true,
		"off":    true,
	}

	cmd := parts[0]
	if !validCommands[cmd] {
		http.Error(w, "Invalid command", http.StatusBadRequest)
		return
	}
	if cmd == "off" {
		cmd = "stop"
	}

	// Any new command supersedes a running indefinite spin/strobe.
	stopIndefinite()

	if cmd == "stop" {
		output, err := exec.Command(fsbeaconBin, "stop").CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v\nOutput: %s", err, output), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf("OK: %s\n", output)))
		return
	}

	// No duration means run until /off - start in the background rather
	// than holding this request open for the life of the process.
	if len(parts) < 2 || parts[1] == "" {
		c := exec.Command(fsbeaconBin, cmd)
		if err := c.Start(); err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
		procMu.Lock()
		indefCmd = c
		procMu.Unlock()
		w.Write([]byte(fmt.Sprintf("OK: %s running until /off\n", cmd)))
		return
	}

	output, err := exec.Command(fsbeaconBin, cmd, parts[1]).CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v\nOutput: %s", err, output), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("OK: %s\n", output)))
}

func main() {
	http.HandleFunc("/", beaconHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK\n"))
	})

	port := ":9100"
	log.Printf("Beacon API starting on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
