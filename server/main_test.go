package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/v6/api4"
	sapp "github.com/mattermost/mattermost-server/v6/app"
	"github.com/mattermost/mattermost-server/v6/app/request"
	"github.com/mattermost/mattermost-server/v6/config"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	ss "github.com/mattermost/mattermost-server/v6/store/sqlstore"
	"github.com/mattermost/mattermost-server/v6/store/storetest"
	"github.com/mattermost/mattermost-server/v6/utils"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Run the plugin under test if the server is trying to run us as a plugin.
	value := os.Getenv("MATTERMOST_PLUGIN")
	if value == "Securely message teams, anywhere." {
		plugin.ClientMain(&Plugin{})
		return
	}

	serverpathBytes, err := exec.Command("go", "list", "-f", "'{{.Dir}}'", "-m", "github.com/mattermost/mattermost-server/v6").Output()
	if err != nil {
		panic(err)
	}
	serverpath := string(serverpathBytes)
	serverpath = strings.Trim(strings.TrimSpace(serverpath), "'")
	os.Setenv("MM_SERVER_PATH", serverpath)

	// This actually runs the tests
	status := m.Run()

	os.Exit(status)
}

type PermissionsHelper interface {
	SaveDefaultRolePermissions() map[string][]string
	RestoreDefaultRolePermissions(data map[string][]string)
	RemovePermissionFromRole(permission string, roleName string)
	AddPermissionToRole(permission string, roleName string)
	SetupChannelScheme() *model.Scheme
}

type serverPermissionsWrapper struct {
	api4.TestHelper
}

type TestEnvironment struct {
	T           testing.TB
	Srv         *sapp.Server
	A           *sapp.App
	SqlStore    *ss.SqlStore
	SqlSettings *model.SqlSettings

	Permissions PermissionsHelper

	AdminClient         *model.Client4
	RegularUserClient   *model.Client4
	RegularUser2Client  *model.Client4
	RegularUser3Client  *model.Client4
	UserNotInTeamClient *model.Client4

	BasicTeam  *model.Team
	BasicTeam2 *model.Team
	// BasicPublicChannel *model.Channel
	// BasicPublicChannelPost  *model.Post
	// BasicPrivateChannel *model.Channel
	// BasicPrivateChannelPost *model.Post

	AdminUser            *model.User
	RegularUser          *model.User
	RegularUser2         *model.User
	RegularUser3         *model.User
	RegularUserNotInTeam *model.User
}

func getEnvWithDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

func Setup(t *testing.T) (e *TestEnvironment) {
	// Environment Settings
	driverName := getEnvWithDefault("TEST_DATABASE_DRIVERNAME", "mysql")

	sqlSettings := storetest.MakeSqlSettings(driverName, false)

	// Directories for plugin stuff
	dir := t.TempDir()
	clientDir := t.TempDir()
	pruneDir := path.Join(dir, "prune")
	binaryDir := path.Join(pruneDir, "server", "dist")
	pluginBinary := path.Join(binaryDir, "plugin-"+runtime.GOOS+"-"+runtime.GOARCH)
	pluginManifest := path.Join(pruneDir, "plugin.json")
	assetsDir := path.Join(pruneDir, "assets")

	// Create a test memory store and modify configuration appropriately
	configStore := config.NewTestMemoryStore()
	myconfig := configStore.Get()
	myconfig.PluginSettings.Directory = &dir
	myconfig.PluginSettings.ClientDirectory = &clientDir
        myconfig.PluginSettings.PluginStates[manifest.Id] = &model.PluginState{Enable:true}
	addr := "localhost:9056"
	myconfig.ServiceSettings.ListenAddress = &addr
	myconfig.TeamSettings.MaxUsersPerTeam = model.NewInt(10000)
	myconfig.LocalizationSettings.SetDefaults()
	myconfig.SqlSettings = *sqlSettings
	myconfig.ServiceSettings.SiteURL = model.NewString("http://testprune.zzh.com/")
	_, _, err := configStore.Set(myconfig)
	require.NoError(t, err)

	// Copy ourselves into the correct directory so we are executed.
	currentBinary, err := os.Executable()
	require.NoError(t, err)
	err = utils.CopyFile(currentBinary, pluginBinary)
	require.NoError(t, err)
	err = utils.CopyDir("../assets", assetsDir)
	require.NoError(t, err)

	// Copy the manifest without webapp to the correct directory
	modifiedManifest := model.Manifest{}
	_ = json.NewDecoder(strings.NewReader(manifestStr)).Decode(&modifiedManifest)
	modifiedManifest.Webapp = nil
	manifestJSONBytes, _ := json.Marshal(modifiedManifest)
	err = os.WriteFile(pluginManifest, manifestJSONBytes, 0700)
	require.NoError(t, err)

	// Create a logger to override
	testLogger, err := mlog.NewLogger()
	require.NoError(t, err)

	logCfg, _ := config.MloggerConfigFromLoggerConfig(&myconfig.LogSettings, nil, config.GetLogFileLocation)
	if errCfg := testLogger.ConfigureTargets(logCfg, nil); errCfg != nil {
		panic("failed to configure test logger: " + errCfg.Error())
	}
	testLogger.LockConfiguration()

	// Create a server with our specified options
	err = utils.TranslationsPreInit()
	require.NoError(t, err)

	options := []sapp.Option{
		sapp.ConfigStore(configStore),
		sapp.SetLogger(testLogger),
	}
	server, err := sapp.NewServer(options...)
	require.NoError(t, err)
	api4.Init(server)
	err = server.Start()
	require.NoError(t, err)

	ap := sapp.New(sapp.ServerConnector(server.Channels()))
	sqlstore := ss.New(server.Config().SqlSettings, nil)

	e = &TestEnvironment{
		T:           t,
		Srv:         server,
		A:           ap,
		SqlStore:    sqlstore,
		SqlSettings: sqlSettings,
		Permissions: &serverPermissionsWrapper{
			TestHelper: api4.TestHelper{
				Server: server,
				App:    ap,
			},
		},
	}

	// Cleanup to run after test is complete
	t.Cleanup(func() {
		server.Shutdown()
		TearDown(e)
	})

	return
}

func TearDown(e *TestEnvironment) {
	if e.SqlStore != nil {
		e.SqlStore.Close()
	}
	if e.SqlSettings != nil {
		storetest.CleanupSqlSettings(e.SqlSettings)
	}
}

func (e *TestEnvironment) CreateClients() {
	e.T.Helper()

	userPassword := "Password123!"
	admin, _ := e.A.CreateUser(request.EmptyContext(), &model.User{
		Email:    "pruneadmin@test.com",
		Username: "pruneadmin",
		Password: userPassword,
	})
	e.AdminUser = admin

	user, _ := e.A.CreateUser(request.EmptyContext(), &model.User{
		Email:    "pruneuser@test.com",
		Username: "pruneuser",
		Password: userPassword,
	})
	e.RegularUser = user

	user2, _ := e.A.CreateUser(request.EmptyContext(), &model.User{
		Email:    "pruneuser2@test.com",
		Username: "pruneuser2",
		Password: userPassword,
	})
	e.RegularUser2 = user2

	user3, _ := e.A.CreateUser(request.EmptyContext(), &model.User{
		Email:    "pruneuser3@test.com",
		Username: "pruneuser3",
		Password: userPassword,
	})
	e.RegularUser3 = user3

	notInTeam, _ := e.A.CreateUser(request.EmptyContext(), &model.User{
		Email:    "pruneusernotinteam@test.com",
		Username: "pruneusenotinteam",
		Password: userPassword,
	})
	e.RegularUserNotInTeam = notInTeam

	siteURL := "http://localhost:9056"

	serverAdminClient := model.NewAPIv4Client(siteURL)
	_, _, err := serverAdminClient.Login(admin.Email, userPassword)
	require.NoError(e.T, err)
	e.AdminClient = serverAdminClient

	serverClient := model.NewAPIv4Client(siteURL)
	_, _, err = serverClient.Login(user.Email, userPassword)
	require.NoError(e.T, err)
	e.RegularUserClient = serverClient

	serverClient2 := model.NewAPIv4Client(siteURL)
	_, _, err = serverClient2.Login(user2.Email, userPassword)
	require.NoError(e.T, err)
	e.RegularUser2Client = serverClient2

	serverClient3 := model.NewAPIv4Client(siteURL)
	_, _, err = serverClient3.Login(user2.Email, userPassword)
	require.NoError(e.T, err)
	e.RegularUser3Client = serverClient3

	serverClientNotInTeam := model.NewAPIv4Client(siteURL)
	_, _, err = serverClientNotInTeam.Login(notInTeam.Email, userPassword)
	require.NoError(e.T, err)
	e.UserNotInTeamClient = serverClientNotInTeam

}

func (e *TestEnvironment) CreateBasicServer() {
	e.T.Helper()

	team, _, err := e.AdminClient.CreateTeam(&model.Team{
		DisplayName: "basic",
		Name:        "basic",
		Email:       "success+prune@simulator.amazonses.com",
		Type:        model.TeamOpen,
	})
	require.NoError(e.T, err)

	_, _, err = e.AdminClient.AddTeamMember(team.Id, e.RegularUser.Id)
	require.NoError(e.T, err)
	_, _, err = e.AdminClient.AddTeamMember(team.Id, e.RegularUser2.Id)
	require.NoError(e.T, err)
	_, _, err = e.AdminClient.AddTeamMember(team.Id, e.RegularUser3.Id)
	require.NoError(e.T, err)

	// pubChannel, _, err := e.AdminClient.CreateChannel(&model.Channel{
	// 	DisplayName: "testpublic1",
	// 	Name:        "testpublic1",
	// 	Type:        model.ChannelTypeOpen,
	// 	TeamId:      team.Id,
	// })
	// require.NoError(e.T, err)

	// pubPost, _, err := e.ServerAdminClient.CreatePost(&model.Post{
	// 	UserId:    e.AdminUser.Id,
	// 	ChannelId: pubChannel.Id,
	// 	Message:   "this is a public channel post by a system admin",
	// })
	// require.NoError(e.T, err)

	// _, _, err = e.AdminClient.AddChannelMember(pubChannel.Id, e.RegularUser.Id)
	// require.NoError(e.T, err)

	// privateChannel, _, err := e.AdminClient.CreateChannel(&model.Channel{
	// 	DisplayName: "testprivate1",
	// 	Name:        "testprivate1",
	// 	Type:        model.ChannelTypePrivate,
	// 	TeamId:      team.Id,
	// })
	// require.NoError(e.T, err)

	// privatePost, _, err := e.ServerAdminClient.CreatePost(&model.Post{
	// 	UserId:    e.AdminUser.Id,
	// 	ChannelId: privateChannel.Id,
	// 	Message:   "this is a private channel post by a system admin",
	// })
	// require.NoError(e.T, err)

	e.BasicTeam = team
	// e.BasicPublicChannel = pubChannel
	// e.BasicPublicChannelPost = pubPost
	// e.BasicPrivateChannel = privateChannel
	// e.BasicPrivateChannelPost = privatePost

	// Add a second team to test cross-team features
	team2, _, err := e.AdminClient.CreateTeam(&model.Team{
		DisplayName: "second team",
		Name:        "second-team",
		Email:       "success+prune@simulator.amazonses.com",
		Type:        model.TeamOpen,
	})
	require.NoError(e.T, err)

	_, _, err = e.AdminClient.AddTeamMember(team2.Id, e.RegularUser.Id)
	require.NoError(e.T, err)

	e.BasicTeam2 = team2
}

func (e *TestEnvironment) CreateBasic() {
	e.CreateClients()
	e.CreateBasicServer()
}

// // TestTestFramework If this is failing you know the break is not exclusively in your test.
// func TestTestFramework(t *testing.T) {
// 	e := Setup(t)
// 	e.CreateBasic()
// }
