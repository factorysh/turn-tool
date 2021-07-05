package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(restCmd)
}

var restCmd = &cobra.Command{
	Use:   "rest",
	Short: "REST server API auth",
	Long:  `Build an ephemeral auth`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("give me a secret")
		}
		login, password, err := buildRestPasswor("alice", []byte(args[0]), 2*time.Hour)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n%s\n", login, password)
		return nil
	},
}
