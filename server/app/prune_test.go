package app_test

import (
	"testing"

	"github.com/ericzzh/mattermost-plugin-prune/server/app"
	mock_prune "github.com/ericzzh/mattermost-plugin-prune/server/app/mocks"
	mock_bot "github.com/ericzzh/mattermost-plugin-prune/server/bot/mocks"
	"github.com/ericzzh/mattermost-plugin-prune/server/config"
	"github.com/ericzzh/mattermost-plugin-prune/server/config/mocks"
	gomock "github.com/golang/mock/gomock"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneStart(t *testing.T) {

	ctrl := gomock.NewController(t)

	configMock := mock_config.NewMockService(ctrl)
	configMock.EXPECT().GetConfiguration().Return(
		&config.Configuration{
			Policy: `
                   settings:
                       days_of_prune:
                           value: 1
                       days_of_deleted: 
                           value: 1
                       only_thread: 
                           value: false
                `},
	).AnyTimes()

	pluginAPI := &plugintest.API{}
	client := pluginapi.NewClient(pluginAPI, &plugintest.Driver{})

	var opt app.Options
	mockPruneStore := mock_prune.NewMockPruneStore(ctrl)
	mockPruneStore.EXPECT().CutRoots("ch1-1_id", gomock.AssignableToTypeOf(opt)).Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().CutThreads("ch1-1_id", gomock.AssignableToTypeOf(opt)).Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().CutDeletedRoots("ch1-1_id", gomock.AssignableToTypeOf(opt)).Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().SweepThreads("ch1-1_id").Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().SweepReactions("ch1-1_id").Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().SweepPreferences("ch1-1_id").Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().SweepFileInfos("ch1-1_id").Return(1, nil).AnyTimes()
        var fm app.FilesMap
	mockPruneStore.EXPECT().SweepFiles("ch1-1_id", gomock.AssignableToTypeOf(fm)).Return(1, nil).AnyTimes()
	mockPruneStore.EXPECT().GetFilesMap().Return(app.FilesMap{}, nil).AnyTimes()

	mockTeamsData(
		map[string][]string{
			"tm1": {"ch1-1"},
		},
		pluginAPI, mockPruneStore)

	mockUsersData(
		map[string][]string{
			"user1": {"userch1_2_id"},
			"user2": {"userch1_2_id"},
		}, pluginAPI)

	pluginAPI.On("GetUsersInChannel", "userch1_2_id", model.ChannelSortByUsername, 1, 1000).Return([]*model.User{
		{Username: "user1"}, {Username: "user2"},
	}, nil)

        loggerMock := mock_bot.NewMockLogger(ctrl) 
        loggerMock.EXPECT().Debugf(gomock.Any()).AnyTimes()

        PosterMock := mock_bot.NewMockPoster(ctrl) 

	pse := app.NewPruneService(mockPruneStore, client, configMock, loggerMock, PosterMock)
	res, err := pse.Start()

	require.NoError(t, err)
	exp := app.Result{
		Teams: app.TeamsRes{
			"tm1_id": app.TeamRes{
				Id:    "tm1_id",
				Name:  "tm1",
				Error: nil,
				Channels: app.ChannelsRes{
					"ch1-1_id": app.ChannelRes{
						Id:    "ch1-1_id",
						Name:  "ch1-1",
						Error: nil,
						Stats: app.Stats{
							CutThreads:       1,
							CutRoots:         1,
							CutDeletedRoots:  1,
							SweepThreads:     1,
							SweepReactions:   1,
							SweepPreferences: 1,
							SweepFileInfos:   1,
							SweepFiles:       1,
						},
					},
				},
			},
		},
		Users: app.UsersRes{
			"user1_id": app.UserRes{
				Id:    "user1_id",
				Names: []string{"user1", "user2"},
				Error: nil,
				Stats: app.Stats{
					CutThreads:       1,
					CutRoots:         1,
					CutDeletedRoots:  1,
					SweepThreads:     1,
					SweepReactions:   1,
					SweepPreferences: 1,
					SweepFileInfos:   1,
					SweepFiles:       1,
				},
			},
		},
	}
	assert.Equal(t, exp, res)
}
