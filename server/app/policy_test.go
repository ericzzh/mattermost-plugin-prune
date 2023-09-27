package app_test

import (
	"fmt"
	"net/http"
	"testing"

	app "github.com/ericzzh/mattermost-plugin-prune/server/app"
	mock_prune "github.com/ericzzh/mattermost-plugin-prune/server/app/mocks"
	gomock "github.com/golang/mock/gomock"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initializeAPI(t *testing.T) (*plugintest.API, app.PruneStore) {
	pluginAPI := &plugintest.API{}
	pluginAPI.On("GetTeamByName", "tm1").Return(&model.Team{Id: "tm1Id"}, nil)
	pluginAPI.On("GetTeamByName", "tm2").Return(nil, &model.AppError{StatusCode: http.StatusNotFound})

	pluginAPI.On("GetChannelByName", "tm1Id", "ch1", true).Return(&model.Channel{}, nil)
	pluginAPI.On("GetChannelByName", "tm1Id", "ch2", true).Return(nil, &model.AppError{StatusCode: http.StatusNotFound})

	pluginAPI.On("GetUserByUsername", "user1").Return(&model.User{}, nil)
	pluginAPI.On("GetUserByUsername", "user2").Return(nil, &model.AppError{StatusCode: http.StatusNotFound})

	ctrl := gomock.NewController(t)
	mockPruneStore := mock_prune.NewMockPruneStore(ctrl)

	return pluginAPI, mockPruneStore
}

func TestPolicyLoad(t *testing.T) {
	pluginAPI, mockPruneStore := initializeAPI(t)
	client := pluginapi.NewClient(pluginAPI, &plugintest.Driver{})

	t.Run("load-from-yaml", func(t *testing.T) {
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)

		yaml := `
                   settings:
                       days_of_prune:
                           value: 30
                       days_of_deleted: 
                           value: 1
                       only_thread: 
                           value: false
                   teams:
                       tm1:
                           settings:
                               days_of_prune:
                                   value: 31
                               days_of_deleted:
                                   value: 2
                               only_thread:
                                   value: true
                           channels:
                               ch1:
                                   settings:
                                       days_of_prune: 
                                           value: 33
                                       days_of_deleted: 
                                           value: 3
                                       only_thread:
                                           value: false
                   users:
                       channels:
                           user1:
                               settings:
                                   days_of_prune: 
                                       value: 34
                                   days_of_deleted: 
                                       value: 4
                                   only_thread: 
                                       value: false
      `
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		assert.Equal(t, app.Policy{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				PersistDeleted:   &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
			Teams: app.Teams{
				"tm1": &app.Team{
					Settings: &app.Settings{
						PersistNormal:    &app.Switch{false},
						PersistDeleted:   &app.Switch{false},
						DaysOfPrune:      &app.Days{31},
						DaysOfDeleted:    &app.Days{2},
						OnlyThreadPruned: &app.Switch{true},
					},
					Channels: app.Channels{
						"ch1": &app.Channel{
							Settings: &app.Settings{
								PersistNormal:    &app.Switch{false},
								PersistDeleted:   &app.Switch{false},
								DaysOfPrune:      &app.Days{33},
								DaysOfDeleted:    &app.Days{3},
								OnlyThreadPruned: &app.Switch{false},
							},
						},
					},
				},
			},
			Users: &app.Users{
				Channels: app.Channels{
					"user1": &app.Channel{
						Settings: &app.Settings{
							PersistNormal:    &app.Switch{false},
							PersistDeleted:   &app.Switch{false},
							DaysOfPrune:      &app.Days{34},
							DaysOfDeleted:    &app.Days{4},
							OnlyThreadPruned: &app.Switch{false},
						},
					},
				},
			},
		}, policyctrl.GetPolicy())
	})

	t.Run("load-error", func(t *testing.T) {
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)

		yaml := `
                   teams:
                       tm2:
                           settings:
                               days_of_prune: 
                                   value: 30
                               days_of_deleted: 
                                   value: 1
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w team:tm2", app.ErrNoTeam).Error())
		yaml = `
                   teams:
                       tm1:
                           channels:
                               ch2:
                                   settings:
                                       days_of_prune: 
                                           value: 30
                                       days_of_deleted: 
                                           value: 1
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w channel:ch2", app.ErrNoChannel).Error())
		yaml = `
                   users:
                       channels:
                           user2:
                               settings:
                                   days_of_prune: 
                                       value: 30
                                   days_of_deleted: 
                                       value: 1
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w user:user2", app.ErrNoUser).Error())

		yaml = `
                   teams:
                       tm1:
                           setttings:
                               days_of_prune: 
                                   value: 30
                               days_of_deleted: 
                                   value: 1
                `
		assert.Error(t, policyctrl.LoadFromYaml(yaml))

		yaml = `
                   users:
                       channels:
                           user1:''
                `
		assert.Error(t, policyctrl.LoadFromYaml(yaml))

	})
	t.Run("load-nil", func(t *testing.T) {
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		yaml := ``
		require.NoError(t, policyctrl.LoadFromYaml(yaml))
		assert.Equal(t, app.Policy{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{true},
				OnlyThreadPruned: &app.Switch{false},
			},
			Teams: nil,
			Users: nil,
		}, policyctrl.GetPolicy())

		yaml = `
                   settings:
                       days_of_prune: 
                           value: 30
                       days_of_deleted: 
                           value: 1
                       only_thread: 
                           value: true
                `
		require.NoError(t, policyctrl.LoadFromYaml(yaml))
		assert.Equal(t, app.Policy{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				PersistDeleted:   &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{true},
			},
			Teams: nil,
			Users: nil,
		}, policyctrl.GetPolicy())
	})
	t.Run("load-mandantory-check", func(t *testing.T) {
		//only root level
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		yaml := `
                   settings:
                       days_of_deleted: 
                           value: 0
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted root", app.ErrRequired).Error())

		yaml = `
                   settings:
                       persist_deleted:
                           value: true
                       days_of_deleted: 
                           value: 1
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted, persist_deleted root", app.ErrConfilct).Error())

		yaml = `
                   settings:
                       persist_deleted:
                           value: false
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted, persist_deleted root", app.ErrConfilct).Error())

		yaml = `
                   settings:
                       days_of_prune: 
                           value: 0
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_prune root", app.ErrRequired).Error())

		yaml = `
                   settings:
                       persist_normal:
                           value: true
                       days_of_prune: 
                           value: 1
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_prune, persist_normal root", app.ErrConfilct).Error())

		yaml = `
                   settings:
                       persist_normal:
                           value: false
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_prune, persist_normal root", app.ErrConfilct).Error())

		yaml = `
                   teams:
                       tm1:
                           settings:
                               days_of_deleted: 
                                   value: 0
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted team:tm1", app.ErrRequired).Error())

		yaml = `
                   teams:
                       tm1:
                           settings:
                               days_of_prune: 
                                   value: 31
                               days_of_deleted: 
                                   value: 30
                           channels:
                               ch1:
                                   settings:
                                       days_of_deleted: 
                                           value: 0
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted channel:ch1", app.ErrRequired).Error())

		yaml = `
                   users:
                       channels:
                           user1:
                               settings:
                                   days_of_deleted: 
                                       value: 0
                `
		assert.EqualError(t, policyctrl.LoadFromYaml(yaml), fmt.Errorf("%w field:days_of_deleted user:user1", app.ErrRequired).Error())

	})

	t.Run("load-decimals", func(t *testing.T) {
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)

		yaml := fmt.Sprintf(`
                        settings:
                            days_of_prune:
                                value: %.10f
      `, 1.0/(24*3600))
		assert.Nil(t, policyctrl.LoadFromYaml(yaml))
	})
}

func TestPolicyExpandChannels(t *testing.T) {

	pluginAPI := &plugintest.API{}

	ctrl := gomock.NewController(t)
	mockPruneStore := mock_prune.NewMockPruneStore(ctrl)

	// pluginAPI.On("GetChannelsForTeamForUser", tmId, "", true).Return(chsmock, nil)
	mockTeamsData(
		map[string][]string{
			"tm10": {"ch10-1"},
			"tm11": {"ch11-1", "ch11-2", "ch11-3", "ch11-4", "ch11-5", "ch11-6"},
			"tm12": {"ch12-1"},
			"tm13": {"ch13-1"},
			"tm14": {"ch14-1"},
			"tm15": {"ch15-1"},
			"tm16": {"ch16-1"},
			"tm17": {"ch17-1"},
		},
		pluginAPI, mockPruneStore)

	mockUsersData(
		map[string][]string{
			"user10": {"userch10-1_id"},
			"user11": {"userch11-1_id"},
			"user12": {"userch12-1_id"},
			"user13": {"userch13-1_id"},
			"user14": {"userch14-1_id"},
		}, pluginAPI)

	client := pluginapi.NewClient(pluginAPI, &plugintest.Driver{})

	t.Run("root-not-nil", func(t *testing.T) {
		yaml := `
                   settings:
                       days_of_prune:
                           value: 30
                       days_of_deleted:
                           value: 1
                       only_thread:
                           value: false
                   teams:
                       tm10:
                           settings:
                               persist_normal:
                                   value: true
                       tm16:
                           settings:
                               persist_normal:
                                   value: true
                               persist_deleted:
                                   value: false
                               days_of_deleted:
                                   value: 2
                       tm13:
                           settings:
                               persist_deleted:
                                   value: true
                       tm17:
                           settings:
                               persist_deleted:
                                   value: true
                               persist_normal:
                                   value: false
                               days_of_prune:
                                   value: 31
                       tm12:
                           settings:
                               persist_normal:
                                   value: true
                               persist_deleted:
                                   value: true
                       tm14:
                           settings:
                               days_of_prune:
                                   value: 40
                               days_of_deleted:
                                   value: 5
                               only_thread:
                                   value:  true
                       tm11:
                           channels:
                               ch11-1:
                                   settings:
                                       persist_normal:
                                           value: true
                                       persist_deleted:
                                           value: true
                               ch11-2:
                                   settings:
                                       persist_normal:
                                           value: true
                               ch11-3:
                                   settings:
                                       persist_deleted:
                                           value: true
                               ch11-4:
                                   settings:
                                       only_thread:
                                           value: true
                               ch11-5:
                                   settings:
                                       days_of_prune:
                                           value: 33
                                       days_of_deleted:
                                           value: 3
                   users:
                       channels:
                           user10:
                               settings:
                                   persist_normal:
                                       value: true
                                   persist_deleted:
                                       value: true
                           user13:
                               settings:
                                   persist_deleted:
                                       value: true
                           user14:
                               settings:
                                   persist_normal:
                                       value: true
                           user11:
                               settings:
                                   days_of_prune:
                                       value: 34
                                   days_of_deleted:
                                       value: 4
                                   only_thread:
                                       value:  true
      `
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err := policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm10_id"].Channels["ch10-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{2},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm16_id"].Channels["ch16-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm13_id"].Channels["ch13-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{31},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm17_id"].Channels["ch17-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{40},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{5},
				OnlyThreadPruned: &app.Switch{true},
			},
		}, expanded.Teams["tm14_id"].Channels["ch14-1_id"])

		assert.NotContains(t, expanded.Teams["tm11_id"].Channels, "ch11-1_id")

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm11_id"].Channels["ch11-2_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm11_id"].Channels["ch11-3_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{true},
			},
		}, expanded.Teams["tm11_id"].Channels["ch11-4_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{3},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{33},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm11_id"].Channels["ch11-5_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm15_id"].Channels["ch15-1_id"])

		assert.NotContains(t, expanded.Teams, "tm12_id")

		assert.NotContains(t, expanded.Users.Channels, "userch10-1_id")

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{34},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{4},
				OnlyThreadPruned: &app.Switch{true},
			},
		}, expanded.Users.Channels["userch11-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch12-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch13-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch14-1_id"])
	})

	t.Run("root-nil-users-setting", func(t *testing.T) {
		yaml := `
                   users:
                       settings:
                           days_of_prune:
                               value: 30
                           days_of_deleted:
                               value: 1
                           only_thread:
                               value: false
                       channels:
                           user13:
                               settings:
                                   persist_deleted:
                                       value: true
                           user14:
                               settings:
                                   persist_normal:
                                       value: true
                           user10:
                               settings:
                                   only_thread:
                                       value: true
      `
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err := policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)
		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch13-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{true},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch14-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{true},
			},
		}, expanded.Users.Channels["userch10-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch11-1_id"])

	})

	t.Run("root-nil-partial", func(t *testing.T) {
		yaml := `
                       settings:
                           days_of_prune:
                               value: 30
                `
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err := policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)
		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm14_id"].Channels["ch14-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{true},
				PersistNormal:    &app.Switch{false},
				DaysOfPrune:      &app.Days{30},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch10-1_id"])

		yaml = `
                       settings:
                           days_of_deleted:
                               value: 1
                `
		policyctrl = app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err = policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)
		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				PersistNormal:    &app.Switch{true},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Teams["tm14_id"].Channels["ch14-1_id"])

		assert.Equal(t, &app.Channel{
			Settings: &app.Settings{
				PersistDeleted:   &app.Switch{false},
				DaysOfDeleted:    &app.Days{1},
				PersistNormal:    &app.Switch{true},
				OnlyThreadPruned: &app.Switch{false},
			},
		}, expanded.Users.Channels["userch10-1_id"])

		yaml = `
                       settings:
                           only_thread:
                               value: true
                `
		policyctrl = app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err = policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)
		assert.Nil(t, expanded.Teams)
		assert.Nil(t, expanded.Users)
	})

	t.Run("root-nil", func(t *testing.T) {
		yaml := ``
		policyctrl := app.NewPolicyService(client, nil, mockPruneStore)
		require.Nil(t, policyctrl.LoadFromYaml(yaml))
		expanded, err := policyctrl.ExpandPolicyAndNormalize()
		require.NoError(t, err)
		assert.Nil(t, expanded.Teams)
		assert.Nil(t, expanded.Users)
	})

}

func mockUsersData(mockdata map[string][]string, pluginAPI *plugintest.API) {
	users := []*model.User{}

	for usr, chs := range mockdata {
		usrId := usr + "_id"
		pluginAPI.On("GetUserByUsername", usr).Return(&model.User{
			Id: usrId,
		}, nil)

		for _, chid := range chs {
			pluginAPI.On("GetChannelsForTeamForUser", "", usrId, true).Return([]*model.Channel{
				{
					Id:   chid,
					Type: model.ChannelTypeDirect,
				},
			}, nil)

		}

		users = append(users, &model.User{
			Id:       usrId,
			Username: usr,
		})
	}

	pluginAPI.On("GetUsers", &model.UserGetOptions{}).Return(users, nil)
}

func mockTeamsData(tms map[string][]string, pluginAPI *plugintest.API, mockPruneStore *mock_prune.MockPruneStore) {
	teams := []*model.Team{}

	for tm, chs := range tms {

		tmId := tm + "_id"
		pluginAPI.On("GetTeamByName", tm).Return(&model.Team{Id: tmId}, nil)

		chsmock := []*model.Channel{}
		for _, ch := range chs {

			chsmock = append(chsmock,
				&model.Channel{
					Id:   ch + "_id",
					Name: ch,
				})

			pluginAPI.On("GetChannelByName", tmId, ch, true).Return(&model.Channel{
				Id: ch + "_id",
			}, nil)
		}

		teams = append(teams, &model.Team{
			Id:   tm + "_id",
			Name: tm,
		})

		mockPruneStore.EXPECT().GetAllChannelsForTeam(tmId).DoAndReturn(
			func(tmId string) ([]*model.Channel, error) {
				return chsmock, nil
			}).AnyTimes()

	}

	pluginAPI.On("GetTeams").Return(teams, nil)
}
