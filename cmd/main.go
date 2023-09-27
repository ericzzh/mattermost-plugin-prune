package main

import (
	"fmt"
	"os"
	// "path/filepath"

	"github.com/mattermost/mattermost-server/v6/cmd/mattermost/commands"
	"github.com/pkg/errors"
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

func pruneCmdF(command *cobra.Command, args []string) (e error) {
	wd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "Prune: Can't get current work directory")
	}
        // MM_WD, _  := filepath.Abs("~/go/src/mattermost-server")
        MM_WD := "/Users/zzh/go/src/mattermost-server"
	err = os.Chdir(MM_WD)
	if err != nil {
		return errors.Wrapf(err, "Prune: Can't change current work directory to %s", MM_WD)
	}
	defer func() {
		err = os.Chdir(wd)
	if err != nil {
		e = errors.Wrapf(err, "Prune: Can't change back to  work directory to %s", wd)
	}
		fmt.Printf("Changed back to working dir: %s", wd)
	}()

	fmt.Printf("Changed to working dir: %s", MM_WD)

	a, err := commands.InitDBCommandContextCobra(command)
	if err != nil {
		return err
	}
	defer a.Srv().Shutdown()

	chs, _ := a.Srv().Store.Channel().GetAll("5tfjpj5m8jdybbct11qy6idpih")

	for _, ch := range chs {
		fmt.Printf("channel: %s\n", ch.Name)
	}
	return nil
}

func main() {

	if err := Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
