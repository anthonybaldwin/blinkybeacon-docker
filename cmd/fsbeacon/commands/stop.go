package commands

import (
	"fmt"
	"github.com/duckfullstop/blinkybeacon/pkg/fsbeacon"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Turns the beacon off.",
	Args:  cobra.NoArgs,
	RunE:  handleStopBeacon,
}

func handleStopBeacon(_ *cobra.Command, _ []string) (err error) {
	var d fsbeacon.Beacon
	d, err = fsbeacon.OpenFarmBeacon()
	if err != nil {
		return
	}
	defer d.Close()

	fmt.Print("Stopping beacon.")
	return d.Stop()
}
