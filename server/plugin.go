package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botID string
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func (p *Plugin) OnActivate() error {

	botID, ensureBotError := p.Helpers.EnsureBot(&model.Bot{
		Username:    "prune",
		DisplayName: "Prune Plugin Bot",
		Description: "A bot account created by the prune plugin.",
	})
	if ensureBotError != nil {
		return errors.Wrap(ensureBotError, "failed to ensure demo bot.")
	}

	p.botID = botID

	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          "prune",
		AutoComplete:     true,
		AutoCompleteHint: "period",
		AutoCompleteDesc: "prune channel's posts",
		AutocompleteData: getCommandPruneAutocompleteData(),
	}); err != nil {
		return errors.Wrapf(err, "failed to register %s command", "prune")
	}

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")

        if trigger != "prune"{

		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown command: " + args.Command),
		}, nil
        }


        return p.executePruneCommand(args),nil
}

func (p *Plugin) executePruneCommand(args *model.CommandArgs) *model.CommandResponse {

          
	return &model.CommandResponse{}
}


func  getCommandPruneAutocompleteData()*model.AutocompleteData{
       
	command := model.NewAutocompleteData("prune", "", "prune channel posts and files")

	command.AddNamedTextArgument("period", "input prune period(seconds)", "", "", true)

	return command
}

