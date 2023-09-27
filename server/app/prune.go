package app

import (
	"time"

	// sq "github.com/Masterminds/squirrel"
	"github.com/ericzzh/mattermost-plugin-prune/server/bot"
	"github.com/ericzzh/mattermost-plugin-prune/server/config"
	pluginapi "github.com/mattermost/mattermost-plugin-api"

	// "github.com/mattermost/mattermost-server/v6/app"
	"github.com/mattermost/mattermost-server/v6/model"
	// "github.com/mattermost/mattermost-server/v6/store/sqlstore"
	"github.com/pkg/errors"
)

type PruneService interface {
	Start() (*Result, error)
}

type Options struct {
	Before int64
}

type File struct {
	Path string
	Size int
}
type PruneStore interface {
	CutThreads(chid string, opt Options) ([]*model.Post, error)
	CutRoots(chid string, opt Options) ([]*model.Post, error)
	CutDeletedRoots(chid string, opt Options) ([]*model.Post, error)
	SweepThreads() ([]*model.Post, error)
	SweepReactions() ([]*model.Reaction, error)
	SweepPreferences() ([]*model.Preference, error)
	SweepFileInfos() ([]*model.FileInfo, error)
	SweepFiles() ([]File, error)

	//copied from mattermost-server channel store( MM_ChannelStore_GetAll)
	GetAllChannelsForTeam(teamID string) ([]*model.Channel, error)
}

type pruneService struct {
	// app           *app.App
	// srv           *app.Server
	// sqlstore      *sqlstore.SqlStore
	logger        bot.Logger
	poster        bot.Poster
	pruneStore    PruneStore
	pluginAPI     *pluginapi.Client
	configService config.Service
}

type pruneObject struct {
	*pruneService
	expanded    *ExpanedPolicyWithId
	baseTime    time.Time
	allPostsMap map[string]*model.Post
	res         Result
}

type StatsFile struct {
	Count int
	Size  int
}

type Stats struct {
	CutThreads       int
	CutRoots         int
	CutDeletedRoots  int
	SweepThreads     int
	SweepReactions   int
	SweepPreferences int
	SweepFileInfos   int
	SweepFilesPath   StatsFile
	SweepFilesThumb  StatsFile
	SweepFilesPrev   StatsFile
}

type ChannelRes struct {
	Id    string
	Name  string
	Error error
	Stats Stats
}

type ChannelsRes map[string]ChannelRes

type TeamRes struct {
	Id       string
	Name     string
	Error    error
	Channels ChannelsRes
}

type TeamsRes map[string]TeamRes

type UserRes struct {
	Id    string
	Names []string
	Error error
	Stats Stats
}

type UsersRes map[string]UserRes

type Sweep struct {
	Error error
	Count int
	Left  int
}

type SweepFiles struct {
	Error     error
	Path      StatsFile
	LeftPath  StatsFile
	Thumbnail StatsFile
	LeftThumb StatsFile
	Preview   StatsFile
	LeftPrev  StatsFile
}

type Result struct {
	Teams            TeamsRes
	Users            UsersRes
	AllTeamsChannels ChannelsRes
	AllUsersChannels UsersRes
	Sweeps           map[string]Sweep
}

func NewPruneService(ps PruneStore, api *pluginapi.Client, cl config.Service, logger bot.Logger, poster bot.Poster) PruneService {

	// s := a.Srv()
	// st, _ := json.Marshal(s.Config().SqlSettings)
	// mlog.Debug(string(st))

	// ss := sqlstore.New(s.Config().SqlSettings, nil)
	//
	// c := cl.GetConfiguration()

	return &pruneService{
		logger:        logger,
		poster:        poster,
		pruneStore:    ps,
		pluginAPI:     api,
		configService: cl,
	}

}

func (p *pruneService) Start() (*Result, error) {
	p.logger.Debugf("Prune: starting prune service. ")
	po, err := p.newPruneObject()
	if err != nil {
		return nil, err
	}

	po.pruneTeams()
	po.pruneUsers()
	po.sweep()
	return &po.res, nil
}

func (p *pruneService) newPruneObject() (*pruneObject, error) {

	p.logger.Debugf("Prune: creating a prune object. ")

	pl := NewPolicyService(p.pluginAPI, p.configService, p.pruneStore)

	if err := pl.LoadFromConfig(); err != nil {
		return nil, errors.Wrapf(err, "Prune: load from config error.")
	}

	ex, err := pl.ExpandPolicyAndNormalize()
	if err != nil {
		return nil, errors.Wrapf(err, "Prune: expand policy error.")
	}

	p.logger.Debugf("Prune: created a prune object. expaned: %#v", ex)

	return &pruneObject{
		pruneService: p,
		expanded:     ex,
		baseTime:     time.Now(),
		allPostsMap:  map[string]*model.Post{},
		res:          Result{},
	}, nil
}

func (po *pruneObject) appendToPosts(src []*model.Post) {
	for _, p := range src {
		po.allPostsMap[p.Id] = p
	}
}

func (po *pruneObject) pruneChannel(id string, settings *Settings) (Stats, error) {

	po.logger.Debugf("Prune: pruning channel. id:%#v.", id)

	stats := Stats{}

	if settings.DaysOfPrune != nil && settings.DaysOfPrune.Value != 0 {
		if settings.OnlyThreadPruned != nil && settings.OnlyThreadPruned.Value == true {
			if posts, err := po.pruneStore.CutThreads(id,
				Options{
					Before: model.GetMillisForTime(po.baseTime.Add(-time.Hour * 24 * time.Duration((*settings).DaysOfPrune.Value))),
				}); err != nil {
				return stats, err
			} else {
				stats.CutThreads = len(posts)
				po.appendToPosts(posts)
			}
		} else {
			if posts, err := po.pruneStore.CutRoots(id,
				Options{
					Before: model.GetMillisForTime(po.baseTime.Add(-time.Hour * 24 * time.Duration((*settings).DaysOfPrune.Value))),
				}); err != nil {
				return stats, err
			} else {
				stats.CutRoots = len(posts)
				po.appendToPosts(posts)
			}
		}
	}

	if settings.DaysOfDeleted != nil && settings.DaysOfDeleted.Value != 0 {
		if settings.OnlyThreadPruned != nil && settings.OnlyThreadPruned.Value == true {
			if posts, err := po.pruneStore.CutRoots(id,
				Options{
					Before: model.GetMillisForTime(po.baseTime.Add(-time.Hour * 24 * time.Duration((*settings).DaysOfDeleted.Value))),
				}); err != nil {
				return stats, err
			} else {
				stats.CutRoots = len(posts)
				po.appendToPosts(posts)
			}
		} else {
			if posts, err := po.pruneStore.CutDeletedRoots(id,
				Options{
					Before: model.GetMillisForTime(po.baseTime.Add(-time.Hour * 24 * time.Duration((*settings).DaysOfDeleted.Value))),
				}); err != nil {
				return stats, err
			} else {
				stats.CutDeletedRoots = len(posts)
				po.appendToPosts(posts)
			}
		}
	}

	return stats, nil
}

func (po *pruneObject) pruneTeams() {

	po.logger.Debugf("Prune: pruning teams.")

	po.res.Teams = TeamsRes{}
	po.res.AllTeamsChannels = ChannelsRes{}

	for id, t := range po.expanded.Teams {
		tr := TeamRes{Id: id}

		tms, err := po.pluginAPI.Team.Get(id)

		if err != nil {
			tr.Error = err
			po.res.Teams[id] = tr
			continue
		}

		tr.Name = tms.Name

		chsRes := ChannelsRes{}
		for cid, c := range t.Channels {

			cr := ChannelRes{Id: cid}

			ch, err := po.pluginAPI.Channel.Get(cid)
			if err != nil {
				cr.Error = err
				chsRes[id] = cr
				continue
			}

			cr.Name = ch.Name

			stats, err := po.pruneChannel(cid, c.Settings)
			if err != nil {
				cr.Error = err
				chsRes[id] = cr
				continue
			}

			cr.Stats = stats

			chsRes[cid] = cr
			po.res.AllTeamsChannels[cid] = cr
		}

		tr.Channels = chsRes
		po.res.Teams[id] = tr
	}
}

func (po *pruneObject) pruneUsers() {
	po.res.Users = UsersRes{}
	po.res.AllUsersChannels = UsersRes{}

	for id, u := range po.expanded.Users.Channels {

		if _, ok := po.res.Users[id]; ok {
			continue
		}

		ur := UserRes{Id: id, Names: []string{}}

		usrs, err := po.pluginAPI.User.ListInChannel(id, model.ChannelSortByUsername, 1, 1000)
		if err != nil {
			ur.Error = err
			po.res.Users[id] = ur
			continue
		}

		for _, usr := range usrs {
			ur.Names = append(ur.Names, usr.Username)
		}

		stats, err := po.pruneChannel(id, u.Settings)
		if err != nil {
			ur.Error = err
			po.res.Users[id] = ur
			continue
		}

		ur.Stats = stats

		po.res.Users[id] = ur
		po.res.AllUsersChannels[id] = ur
	}
}

func (po *pruneObject) calResult(postid string, addone func(Stats)) {
	if p, ok := po.allPostsMap[postid]; ok {
		if r, ok := po.res.AllTeamsChannels[p.ChannelId]; ok {
			addone(r.Stats)
		} else if r, ok := po.res.AllUsersChannels[p.ChannelId]; ok {
			addone(r.Stats)
		}
	}
}
func (po *pruneObject) sweep() {

	sweep := Sweep{}
	if posts, err := po.pruneStore.SweepThreads(); err != nil {
		sweep.Error = err
	} else {
		var moved int
		for _, p := range posts {
			po.calResult(p.RootId, func(s Stats) { s.SweepThreads++; moved++ })
			sweep.Count++
		}
		sweep.Left = sweep.Count - moved
		po.appendToPosts(posts)
	}
	po.res.Sweeps["SweepThreads"] = sweep

	sweep = Sweep{}
	if reactions, err := po.pruneStore.SweepReactions(); err != nil {
		sweep.Error = err
	} else {
		var moved int
		for _, r := range reactions {
			po.calResult(r.PostId, func(s Stats) { s.SweepReactions++; moved++ })
		}
		sweep.Left = sweep.Count - moved
	}
	po.res.Sweeps["SweepReactions"] = sweep

	sweep = Sweep{}
	if prefs, err := po.pruneStore.SweepPreferences(); err != nil {
		sweep.Error = err
	} else {
		var moved int
		for _, pref := range prefs {
			po.calResult(pref.Name, func(s Stats) { s.SweepPreferences++; moved++ })
		}
		sweep.Left = sweep.Count - moved
	}
	po.res.Sweeps["SweepPreferences"] = sweep

	sweep = Sweep{}
	pathMap := map[string]*model.FileInfo{}
	thumbMap := map[string]*model.FileInfo{}
	prevMap := map[string]*model.FileInfo{}
	if fileInfos, err := po.pruneStore.SweepFileInfos(); err != nil {
		sweep.Error = err
	} else {
		var moved int
		for _, fileInfo := range fileInfos {
			po.calResult(fileInfo.PostId, func(s Stats) { s.SweepFileInfos++; moved++ })

			if fileInfo.Path != "" {
				pathMap[fileInfo.Path] = fileInfo
			}

			if fileInfo.ThumbnailPath != "" {
				thumbMap[fileInfo.ThumbnailPath] = fileInfo
			}

			if fileInfo.PreviewPath != "" {
				prevMap[fileInfo.PreviewPath] = fileInfo
			}
		}
		sweep.Left = sweep.Count - moved
	}
	po.res.Sweeps["SweepFileInfos"] = sweep

	sweepFiles := SweepFiles{}
	if files, err := po.pruneStore.SweepFiles(); err != nil {
		sweep.Error = err
	} else {
		var (
			movedpath  StatsFile
			movedthumb StatsFile
			movedprev  StatsFile
		)
		for _, file := range files {
			if f, ok := pathMap[file.Path]; ok {
				sweepFiles.Path.Count++
				sweepFiles.Path.Size += file.Size
				po.calResult(f.PostId, func(s Stats) {
					s.SweepFilesPath.Count++
					movedpath.Count++
					s.SweepFilesPath.Size += file.Size
					movedpath.Size += file.Size
				})
			}

			if f, ok := thumbMap[file.Path]; ok {
				sweepFiles.Thumbnail.Count++
				sweepFiles.Thumbnail.Size += file.Size
				po.calResult(f.PostId, func(s Stats) {
					s.SweepFilesThumb.Count++
					movedthumb.Count++
					s.SweepFilesThumb.Size += file.Size
					movedthumb.Size += file.Size
				})
			}

			if f, ok := prevMap[file.Path]; ok {
				sweepFiles.Preview.Count++
				sweepFiles.Preview.Size++
				po.calResult(f.PostId, func(s Stats) {
					s.SweepFilesPrev.Count++
					movedprev.Count++
					s.SweepFilesPrev.Size += file.Size
					movedprev.Size += file.Size
				})
			}
		}
	}
	po.res.Sweeps["SweepFiles"] = sweep

}
