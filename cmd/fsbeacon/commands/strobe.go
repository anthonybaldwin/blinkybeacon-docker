package commands

import (
	"fmt"
	"github.com/duckfullstop/blinkybeacon/pkg/fsbeacon"
	"github.com/spf13/cobra"
	"time"
)

func init() {
	rootCmd.AddCommand(strobeCmd)
}

var strobeCmd = &cobra.Command{
	Use:   "strobe (seconds)",
	Short: "Flash the beacon for a set length of time. Defaults to 5 seconds.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  handleStrobeBeacon,
}

func handleStrobeBeacon(_ *cobra.Command, args []string) (err error) {
	var runtime time.Duration
	if len(args) > 0 {
		runtime, err = parseRuntime(args[0])
		if err != nil {
			return
		}
	}

	var d fsbeacon.Beacon
	d, err = fsbeacon.OpenFarmBeacon()
	if err != nil {
		return
	}
	defer d.Close()

	// Start flashing the beacon - this starts a routine in the package to ensure the beacon keeps running indefinitely
	if runtime != 0 {
		fmt.Printf("Strobing beacon for %s.", runtime)
	} else {
		fmt.Printf("Strobing beacon - press ^C to stop.")
	}

	err = d.Flash()
	if err != nil {
		return err
	}

	// Wait for the configured runtime (the aforementioned goroutine is making sure the beacon is whipped)
	// or a SIGTERM from somewhere
	if runtime != 0 {
		runWithTimeout(runtime)
	} else {
		runUntilInterrupted()
	}

	// Now stop the beacon with the stop command. The connection gets tidied up by the earlier defer.
	err = d.Stop()
	return err
}
