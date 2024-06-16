package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bryk-io/serve/internal"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"info"},
	Short:   "Show version information",
	Run: func(_ *cobra.Command, _ []string) {
		var components = map[string]string{
			"Version":    internal.CoreVersion,
			"Build code": internal.BuildCode,
			"OS/Arch":    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"Go version": runtime.Version(),
			"Home":       "https://github.com/bryk-io/serve",
			"Release":    conf.ReleaseCode(),
		}
		if internal.BuildTimestamp != "" {
			st, err := time.Parse(time.RFC3339, internal.BuildTimestamp)
			if err == nil {
				components["Release Date"] = st.Format(time.RFC822)
			}
		}
		for k, v := range components {
			fmt.Printf("\033[21;37m%-13s:\033[0m %s\n", k, v)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
