package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/ericzzh/mattermost-plugin-prune/server/app"
	"github.com/ericzzh/mattermost-plugin-prune/server/bot"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/pkg/errors"
)

const helpText = "######  Prune Plugin - Slash Command Help\n" +
	"* `/prune run` - Run a prune. \n" +
	""

// Register is a function that allows the runner to register commands with the mattermost server.
type Register func(*model.Command) error

// RegisterCommands should be called by the plugin to register all necessary commands
func RegisterCommands(registerFunc Register, addTestCommands bool) error {
	return registerFunc(getCommand(addTestCommands))
}

func getCommand(addTestCommands bool) *model.Command {
	return &model.Command{
		Trigger:          "prune",
		DisplayName:      "Prune",
		Description:      "Prune posts and files",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: run",
		AutoCompleteHint: "[command]",
		AutocompleteData: getAutocompleteData(addTestCommands),
	}
}

func getAutocompleteData(addTestCommands bool) *model.AutocompleteData {
	command := model.NewAutocompleteData("prune", "[command]",
		"Available commands: run")

	run := model.NewAutocompleteData("run", "", "Starts a prune run")
	run.AddNamedTextArgument("period", "input prune period(seconds)", "", "", true)
	command.AddCommand(run)

	if addTestCommands {
	}

	return command
}

// Runner handles commands.
type Runner struct {
	context      *plugin.Context
	args         *model.CommandArgs
	pluginAPI    *pluginapi.Client
	logger       bot.Logger
	poster       bot.Poster
	pruneService app.PruneService
}

// NewCommandRunner creates a command runner.
func NewCommandRunner(ctx *plugin.Context,
	args *model.CommandArgs,
	api *pluginapi.Client,
	logger bot.Logger,
	poster bot.Poster,
	ps app.PruneService,
) *Runner {
	return &Runner{
		context:      ctx,
		args:         args,
		pluginAPI:    api,
		logger:       logger,
		poster:       poster,
		pruneService: ps,
	}
}

func (r *Runner) isValid() error {
	if r.context == nil || r.args == nil || r.pluginAPI == nil {
		return errors.New("invalid arguments to command.Runner")
	}
	return nil
}

// Execute should be called by the plugin when a command invocation is received from the Mattermost server.
func (r *Runner) Execute() error {
	if err := r.isValid(); err != nil {
		return err
	}

	split := strings.Fields(r.args.Command)
	command := split[0]
	parameters := []string{}
	cmd := ""
	if len(split) > 1 {
		cmd = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/prune" {
		return nil
	}

	switch cmd {
	case "run":
		r.actionRun(parameters)
	default:
		r.postCommandResponse(helpText)
	}

	return nil
}

func (r *Runner) postCommandResponse(text string) {
	post := &model.Post{
		Message: text,
	}
	r.poster.EphemeralPost(r.args.UserId, r.args.ChannelId, post)
}

func (r *Runner) actionRun(args []string) {
	// period := ""
	// if len(args) > 0 {
	// 	period = args[0]
	// }

	usr, apperr := r.pluginAPI.User.Get(r.args.UserId)
	if apperr != nil {
		err_str, _ := json.MarshalIndent(apperr, "", "\t")
		r.postCommandResponse(fmt.Sprintf("Can't find user. Error: \n %v ", string(err_str)))
		return
	}

	if !strings.Contains(usr.Roles, "system_admin") {
		r.postCommandResponse("You don't have permission to run this command.")
		return
	}

	// wd, err := os.Getwd()
	//
	// if err != nil {
	// 	r.postCommandResponse(errors.Wrapf(err, "Prune: Can't get current work directory").Error())
	// 	return
	// }
	// srvPath := os.Getenv("MM_SERVER_PATH")
	// if srvPath == "" {
	// 	r.postCommandResponse("Please set MM_SERVER_PATH environment variable to you mattermost-server path")
	// 	return
	// }
	//
	// err = os.Chdir(srvPath)
	// if err != nil {
	// 	r.postCommandResponse(errors.Wrapf(err, "Prune: Can't change current work directory to %s", srvPath).Error())
	// 	return
	// }
	// defer os.Chdir(wd)
	//
	// fmt.Printf("Changed to working dir: %s", srvPath)
	// a, err := commands.InitDBCommandContextCobra(&cobra.Command{})
	//
	// if err != nil {
	// 	r.postCommandResponse(fmt.Sprintf("Starting server failed. error: %v", err))
	// 	return
	// }
	// defer a.Srv().Shutdown()
	//
	// p, err := strconv.Atoi(period)
	// if err != nil || p == 0 {
	// 	r.postCommandResponse(fmt.Sprintf("Please input a number"))
	// 	return
	// }

	rs, err := r.pruneService.Start()
	if err != nil {
                txt := fmt.Sprintf("Start Prune Service error. %v", err.Error())
		r.logger.Errorf(txt)
		r.postCommandResponse(txt)
		return
	}

	res, err := json.MarshalIndent(rs, "", "\t")
	if err != nil {
                txt :=fmt.Sprintf("Marshaling statistcis to json has errors. %v", err) 
		r.logger.Errorf(txt)
		r.postCommandResponse(txt)
		return
	}

	r.postCommandResponse(fmt.Sprintf("Pruned sccussfully. \n ```%v```", string(res)))
	return

}
