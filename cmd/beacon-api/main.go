package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	fsbeaconBin = "/usr/local/bin/fsbeacon"

	// How long a superseded run gets to exit after SIGTERM before it's killed.
	stopTimeout = 5 * time.Second
)

// A spin/strobe keeps a CLI process alive to refresh the beacon's keepalive
// for the life of the run, timed or not. Track it so /stop or a new command
// can end it - otherwise its keepalive would override the new state.
type run struct {
	cmd        *exec.Cmd
	output     bytes.Buffer
	err        error
	done       chan struct{}
	superseded bool // set before done is closed
}

var (
	procMu  sync.Mutex
	current *run
)

// stopCurrentLocked terminates the running spin/strobe, if any, and waits for
// it to exit. procMu must be held.
func stopCurrentLocked() {
	if current == nil {
		return
	}
	current.superseded = true
	current.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-current.done:
	case <-time.After(stopTimeout):
		current.cmd.Process.Kill()
		<-current.done
	}
	current = nil
}

// startRun supersedes any running spin/strobe with a new fsbeacon process.
// Stopping and starting happen under one lock so concurrent requests can't
// leave an untracked process behind.
func startRun(args ...string) (*run, error) {
	procMu.Lock()
	defer procMu.Unlock()
	stopCurrentLocked()

	r := &run{cmd: exec.Command(fsbeaconBin, args...), done: make(chan struct{})}
	r.cmd.Stdout = &r.output
	r.cmd.Stderr = &r.output
	if err := r.cmd.Start(); err != nil {
		return nil, err
	}
	// Always reap the process, even if it exits on its own. done is closed
	// before taking procMu, since stopCurrentLocked waits on it holding procMu.
	go func() {
		r.err = r.cmd.Wait()
		close(r.done)
		procMu.Lock()
		if current == r {
			current = nil
		}
		procMu.Unlock()
	}()
	current = r
	return r, nil
}

// parseDuration mirrors the CLI's accepted formats (plain seconds or a Go
// duration string).
func parseDuration(arg string) (time.Duration, error) {
	var d time.Duration
	if seconds, err := strconv.ParseFloat(arg, 64); err == nil {
		// Reject values that don't convert to a valid time.Duration.
		if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > math.MaxInt64/float64(time.Second) {
			return 0, fmt.Errorf("invalid duration %q", arg)
		}
		d = time.Duration(seconds * float64(time.Second))
	} else if d, err = time.ParseDuration(arg); err != nil {
		return 0, fmt.Errorf("invalid duration %q", arg)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive (omit it to run until /off)")
	}
	return d, nil
}

func beaconHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) > 2 {
		http.Error(w, "Usage: /[command]/[duration]", http.StatusBadRequest)
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

	if cmd == "stop" {
		// Hold the lock so a concurrent spin/strobe can't start mid-stop.
		procMu.Lock()
		stopCurrentLocked()
		output, err := exec.Command(fsbeaconBin, "stop").CombinedOutput()
		procMu.Unlock()
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
		if _, err := startRun(cmd); err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf("OK: %s running until /off\n", cmd)))
		return
	}

	d, err := parseDuration(parts[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pass the normalized duration after "--" so it can never be read as a flag.
	run, err := startRun(cmd, "--", d.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	<-run.done
	if run.superseded {
		w.Write([]byte(fmt.Sprintf("OK: %s superseded by a later command\n", cmd)))
		return
	}
	if run.err != nil {
		http.Error(w, fmt.Sprintf("Error: %v\nOutput: %s", run.err, run.output.String()), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("OK: %s\n", run.output.String())))
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
