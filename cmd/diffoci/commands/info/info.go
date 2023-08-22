package info

import (
	"encoding/json"
	"fmt"

	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "info",
		Short:                 "Display diagnostic information",
		Args:                  cobra.NoArgs,
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.Bool("json", false, "Display the result as JSON")
	return cmd
}

type Info struct {
	Backend backend.Info `json:"Backend"`
	Version string       `json:"Version"`
}

func action(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	flagJSON, err := flags.GetBool("json")
	if err != nil {
		return err
	}
	b, err := backendmanager.NewBackend(cmd)
	if err != nil {
		return err
	}
	info := Info{
		Backend: b.Info(),
		Version: version.GetVersion(),
	}

	w := cmd.OutOrStdout()
	if flagJSON {
		b, err := json.MarshalIndent(info, "", "    ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
	} else {
		fmt.Fprintf(w, "Backend: %s\n", info.Backend.Name)
		fmt.Fprintf(w, "Version: %s\n", info.Version)
	}
	return nil
}
