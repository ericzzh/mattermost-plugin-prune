package main

import (
	"fmt"
	"os"

	"github.com/mattermost/mattermost-server/v5/cmd/mattermost/commands"
	"github.com/spf13/cobra"
)

type Command = cobra.Command

func Run(args []string) error {
	RootCmd.SetArgs(args)
	return RootCmd.Execute()
}

var RootCmd = &cobra.Command{
	Use:   "prune",
	Short: "prune expired posts and file",
	RunE:  pruneCmdF,
}

func pruneCmdF(command *cobra.Command, args []string) error {

	a, err := commands.InitDBCommandContextCobra(command)
	if err != nil {
		return err
	}
        chs,_ := a.Srv().Store.Channel().GetChannelsByIds([]string{"h4pt9jnbstyjpmzn44gbshd1cy"},false)

        for _, ch := range chs {
             fmt.Printf("channel: %s\n", ch.Name)
        }
	defer a.Srv().Shutdown()
	return nil
}

func main() {

	if err := Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
