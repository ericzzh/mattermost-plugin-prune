package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/ericzzh/mattermost-plugin-prune/server/app"
	"github.com/ericzzh/mattermost-plugin-prune/server/bot"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

type pruneStore struct {
	pluginAPI    PluginAPIClient
	log          bot.Logger
	store        *SQLStore
	queryBuilder sq.StatementBuilderType
}

func NewPruneStore(pluginAPI PluginAPIClient, log bot.Logger, sqlStore *SQLStore) app.PruneStore {
	newStore := &pruneStore{
		pluginAPI:    pluginAPI,
		log:          log,
		store:        sqlStore,
		queryBuilder: sqlStore.builder,
	}
	return newStore
}

func (ps *pruneStore) CutThreads(chid string, opt app.Options) (int,error) {
	return 0,nil
}

func (ps *pruneStore) CutRoots(chid string, opt app.Options) (int,error) {
	ps.store.execBuilder(ps.store.db,
		ps.queryBuilder.Delete("Posts").Where("CreateAt < ?", opt.Before))
	return 0,nil
}

func (ps *pruneStore) CutDeletedRoots(chid string, opt app.Options) (int,error) {

	return 0,nil
}

func (ps *pruneStore) CutDeletedThreads(chid string, opt app.Options) (int,error) {

	return 0,nil
}

func (ps *pruneStore) SweepThreads(chid string) (int,error) {

	return 0,nil
}

func (ps *pruneStore) SweepReactions(chid string) (int,error) {

	return 0,nil
}
func (ps *pruneStore) SweepPreferences(chid string) (int,error) {

	return 0,nil
}
func (ps *pruneStore) SweepFileInfos(chid string) (int,error) {

	return 0,nil
}
func (ps *pruneStore) SweepFiles(chid string, fm app.FilesMap) (int,error) {

	return 0,nil
}
func (ps *pruneStore) GetFilesMap() (app.FilesMap, error) {
	return nil, nil
}

//refer to mattrermost-server channel-store GetAll
func (ps *pruneStore) GetAllChannelsForTeam(teamId string) ([]*model.Channel, error) {
	data := []*model.Channel{}

	query := ps.store.builder.
		Select("*").
		From("Channels").
		Where(sq.And{
			sq.Eq{"TeamId": teamId},
			sq.Or{
				sq.Eq{"Type": model.ChannelTypeOpen},
				sq.Eq{"Type": model.ChannelTypePrivate},
			}})

	err := ps.store.selectBuilder(ps.store.db, &data, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Channels with teamId=%s", teamId)
	}

	return data, nil
}
