package main

import (
	// "encoding/json"
	"fmt"
	"net/http"

	// "os"
	// "strconv"
	// "strings"
	// "time"

	// "github.com/mattermost/mattermost-server/v6/app"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	// "github.com/mattermost/mattermost-server/v6/shared/i18n"
	// "github.com/mattermost/mattermost-server/v6/utils"
	"github.com/pkg/errors"
	// "github.com/spf13/cobra"

	// "github.com/mattermost/mattermost-plugin-api/cluster"
	// "github.com/ericzzh/mattermost-plugin-prune/server/app"
	"github.com/ericzzh/mattermost-plugin-prune/server/app"
	"github.com/ericzzh/mattermost-plugin-prune/server/bot"
	"github.com/ericzzh/mattermost-plugin-prune/server/command"
	"github.com/ericzzh/mattermost-plugin-prune/server/config"
	pluginapi "github.com/mattermost/mattermost-plugin-api"

	// "github.com/mattermost/mattermost-server/v6/cmd/mattermost/commands"
	"github.com/ericzzh/mattermost-plugin-prune/server/sqlstore"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin
	config    *config.ServiceImpl
	pluginAPI *pluginapi.Client
	bot       *bot.Bot
        pruneService app.PruneService

	// backgroundJob is a job that executes periodically on only one plugin instance at a time
	// backgroundJob *cluster.Job
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func (p *Plugin) OnActivate() error {

	pluginAPIClient := pluginapi.NewClient(p.API, p.Driver)
	p.pluginAPI = pluginAPIClient

	p.config = config.NewConfigService(pluginAPIClient, manifest)

	botID, ensureBotError := pluginAPIClient.Bot.EnsureBot(&model.Bot{
		Username:    "prune",
		DisplayName: "Prune Plugin Bot",
		Description: "A bot account created by the prune plugin.",
	})
	if ensureBotError != nil {
		return errors.Wrap(ensureBotError, "failed to ensure demo bot.")
	}

	err := p.config.UpdateConfiguration(func(c *config.Configuration) {
		c.BotUserID = botID
	})
	if err != nil {
		return errors.Wrapf(err, "failed save bot to config")
	}

	apiClient := sqlstore.NewClient(pluginAPIClient)
	p.bot = bot.New(pluginAPIClient, p.config.GetConfiguration().BotUserID, p.config)

	sqlStore, err := sqlstore.New(apiClient, p.bot)
	if err != nil {
		return errors.Wrapf(err, "failed creating the SQL store")
	}

	if err = command.RegisterCommands(p.API.RegisterCommand, false); err != nil {
		return errors.Wrapf(err, "failed register commands")
	}

        pruneStore := sqlstore.NewPruneStore(apiClient, p.bot, sqlStore)
        p.pruneService = app.NewPruneService(pruneStore, pluginAPIClient, p.config, p.bot, p.bot)

	// if err := p.API.RegisterCommand(&model.Command{
	// 	Trigger:          "prune",
	// 	AutoComplete:     true,
	// 	AutoCompleteHint: "period",
	// 	AutoCompleteDesc: "prune channel's posts",
	// 	AutocompleteData: getCommandPruneAutocompleteData(),
	// }); err != nil {
	// 	return errors.Wrapf(err, "failed to register %s command", "prune")
	// }

	// job, cronErr := cluster.Schedule(
	// 	p.API,
	// 	"BackgroundJob",
	// 	cluster.MakeWaitForRoundedInterval(15*time.Minute),
	// 	p.BackgroundJob,
	// )
	// if cronErr != nil {
	// 	return errors.Wrap(cronErr, "failed to schedule background job")
	// }
	// p.backgroundJob = job
	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	runner := command.NewCommandRunner(c, args, pluginapi.NewClient(p.API, p.Driver), p.bot, p.bot, p.pruneService)

	if err := runner.Execute(); err != nil {
		return nil, model.NewAppError("Prune.ExecuteCommand", "app.command.execute.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return &model.CommandResponse{}, nil

	// trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")
	//
	// if trigger != "prune" {
	//
	// 	return &model.CommandResponse{
	// 		ResponseType: model.CommandResponseTypeEphemeral,
	// 		Text:         fmt.Sprintf("Unknown command: " + args.Command),
	// 	}, nil
	// }
	//
	// return p.executePruneCommand(args), nil
}

// func (p *Plugin) executePruneCommand(args *model.CommandArgs) *model.CommandResponse {
//
// 	usr, apperr := p.API.GetUser(args.UserId)
// 	if apperr != nil {
// 		err_str, _ := json.MarshalIndent(apperr, "", "\t")
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         fmt.Sprintf("Can't find user. Error: \n %v ", string(err_str)),
// 		}
// 	}
//
// 	if !strings.Contains(usr.Roles, "system_admin") {
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         "You don't have permission to run this command.",
// 		}
// 	}
//
// 	wd, err := os.Getwd()
//
// 	if err != nil {
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         errors.Wrapf(err, "Prune: Can't get current work directory").Error(),
// 		}
// 	}
// 	srvPath := os.Getenv("MM_SERVER_PATH")
// 	if srvPath == "" {
//
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         "Please set MM_SERVER_PATH environment variable to you mattermost-server path",
// 		}
// 	}
// 	err = os.Chdir(srvPath)
// 	if err != nil {
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         errors.Wrapf(err, "Prune: Can't change current work directory to %s", srvPath).Error(),
// 		}
// 	}
// 	defer os.Chdir(wd)
//
// 	fmt.Printf("Changed to working dir: %s", srvPath)
// 	a, err := commands.InitDBCommandContextCobra(&cobra.Command{})
// 	// a, err := initDBCommandContext("config.json", true)
//
// 	if err != nil {
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         fmt.Sprintf("Starting server failed. error: %v", err),
// 		}
// 	}
// 	defer a.Srv().Shutdown()
//
// 	split := strings.Fields(args.Command)
// 	arg := split[1]
// 	switch arg {
// 	case "--period":
// 		p, err := strconv.Atoi(split[2])
// 		if err != nil || p == 0 {
// 			return &model.CommandResponse{
// 				ResponseType: model.CommandResponseTypeEphemeral,
// 				Text:         fmt.Sprintf("Please input a number"),
// 			}
// 		}
// 		pr, err := app.NewPrune(a)
// 		if err != nil {
// 			return &model.CommandResponse{
// 				ResponseType: model.CommandResponseTypeEphemeral,
// 				Text:         fmt.Sprintf("Creating prune object error. %v", err.Error()),
// 			}
// 		}
// 		stats, err := pr.PruneAction([]string{args.ChannelId}, nil, time.Duration(p))
// 		if err != nil {
// 			return &model.CommandResponse{
// 				ResponseType: model.CommandResponseTypeEphemeral,
// 				Text:         fmt.Sprintf("Pruning error. %v", err.Error()),
// 			}
// 		}
//
// 		res, err := json.MarshalIndent(stats, "", "\t")
// 		if err != nil {
// 			return &model.CommandResponse{
// 				ResponseType: model.CommandResponseTypeEphemeral,
// 				Text:         fmt.Sprintf("Marshaling statistcis to json has errors. %v", err.Error()),
// 			}
// 		}
//
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         fmt.Sprintf("Pruned sccussfully. \n ```%v```", string(res)),
// 		}
//
// 	default:
//
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         fmt.Sprintf("Unknown named argument %v", arg),
// 		}
// 	}
//
// }

// func getCommandPruneAutocompleteData() *model.AutocompleteData {
//
// 	command := model.NewAutocompleteData("prune", "", "prune channel posts and files")
//
// 	command.AddNamedTextArgument("period", "input prune period(seconds)", "", "", true)
//
// 	return command
// }

// OnConfigurationChange handles any change in the configuration.
func (p *Plugin) OnConfigurationChange() error {
	if p.config == nil {
		return nil
	}

	return p.config.OnConfigurationChange()
}

// // OnConfigurationChange is invoked when configuration changes may have been made.
// func (p *Plugin) OnConfigurationChange() error {
// 	var configuration = new(config.Configuration)
//
// 	// Load the public configuration fields from the Mattermost server configuration.
// 	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
// 		return errors.Wrap(err, "failed to load plugin configuration")
// 	}
//
// 	configuration.setConfiguration(configuration)
//
// 	// return nil
// 	return nil
// }

// // copy from mattermost-server/cmd/mattermost/commands/init.go/initDBCommandContext
// func initDBCommandContext(configDSN string, readOnlyConfigStore bool) (*app.App, error) {
// 	if err := utils.TranslationsPreInit(); err != nil {
// 		return nil, err
// 	}
// 	model.AppErrorInit(i18n.T)
//
// 	s, err := app.NewServer(
// 		app.Config(configDSN, false, readOnlyConfigStore, nil),
// 		app.StartSearchEngine,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	a := app.New(app.ServerConnector(s))
//
// 	if model.BuildEnterpriseReady == "true" {
// 		a.Srv().LoadLicense()
// 	}
//
// 	return a, nil
// }
