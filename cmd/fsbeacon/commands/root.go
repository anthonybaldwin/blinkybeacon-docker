package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var rootCmd = &cobra.Command{
	Use:   "fsbeacon",
	Short: "fsbeacon is a simple control application for controlling Farming Simulator beacons.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// parseRuntime accepts either a plain number of seconds ("2", "2.5")
// or a Go duration string ("2s", "500ms", "1m").
func parseRuntime(arg string) (time.Duration, error) {
	var d time.Duration
	if seconds, err := strconv.ParseFloat(arg, 64); err == nil {
		d = time.Duration(seconds * float64(time.Second))
	} else if d, err = time.ParseDuration(arg); err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive (omit it to run until stopped)")
	}
	return d, nil
}

func runUntilInterrupted() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Print("Caught SIGTERM, exiting.")
}

func runWithTimeout(t time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	select {
	case <-c:
		fmt.Print("Caught SIGTERM, exiting.")
		return
	case <-time.After(t):
		return
	}
}
