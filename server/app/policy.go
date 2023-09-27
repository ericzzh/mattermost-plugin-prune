package app

import (
	"errors"
	"fmt"

	"github.com/ericzzh/mattermost-plugin-prune/server/config"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	// "github.com/mattermost/mattermost-server/v6/app"
	"github.com/mattermost/mattermost-server/v6/model"
	"gopkg.in/yaml.v2"
	// "time"
)

// type SimpleRetention struct {
// 	srv *app.Server
// }
//
// const SIMPLE_RETENTION_KIND_TEAM = "Team"
// const SIMPLE_RETENTION_KIND_CHANNEL = "Channel"
// const SIMPLE_RETENTION_KIND_USER = "User"

type PolicyService interface {
	LoadFromConfig() error
	LoadFromYaml(yamlStr string) error
	ExpandPolicyAndNormalize() (*ExpanedPolicyWithId, error)
        GetPolicy() Policy
}

var (
	ErrNoTeam    = errors.New("can not found team.")
	ErrNoChannel = errors.New("can not found channel.")
	ErrNoUser    = errors.New("can not found user.")
	ErrRequired  = errors.New("require setting.")
	ErrConfilct  = errors.New("conflict setting.")
)

type Days struct {
	Value float32 `yaml:"value"`
}

type Switch struct {
	Value bool `yaml:"value"`
}

var (
	DAY0         = Days{0}
	SWITCH_FALSE = Switch{false}
	SWITCH_TRUE  = Switch{true}
)

/*
   if nil, use Settings 1 level up
   if root's setting is nil, no deletetion is default
   if Persist* = true, then Days* will always be 0. they are paires

   Deleted posts and Non-Deleted posts are seperated.

   if OnlyThreadPruned then prune only thread
   else only roots
*/
type Settings struct {
	PersistNormal  *Switch `yaml:"persist_normal"`
	PersistDeleted *Switch `yaml:"persist_deleted"`
	DaysOfPrune    *Days   `yaml:"days_of_prune"`
	DaysOfDeleted  *Days   `yaml:"days_of_deleted"`
	//OnlyThreadPruned prune only threads.
	//context free of Persist*
	OnlyThreadPruned *Switch `yaml:"only_thread"`
}

type Channel struct {
	Settings *Settings `yaml:"settings,omitempty"`
}
type Channels map[string]*Channel

type Team struct {
	Settings *Settings `yaml:"settings,omitempty"`
	Channels Channels  `yaml:"channels,omitempty"`
}
type Teams map[string]*Team

type Users Team

type Policy struct {
	//nil means using default
	Settings *Settings `yaml:"settings,omitempty"`
	Teams    Teams     `yaml:"teams,omitempty"`
	Users    *Users    `yaml:"users,omitempty"`
}

type ExpanedPolicyWithId Policy

type PolicyCtrl struct {
	pluginAPI     *pluginapi.Client
	Policy        Policy
	configService config.Service
	pruneStore    PruneStore
}

func NewPolicyService(apiClient *pluginapi.Client, cl config.Service, pstore PruneStore) PolicyService {
	return &PolicyCtrl{
		pluginAPI:     apiClient,
		configService: cl,
		pruneStore:    pstore,
	}
}

func (pctl *PolicyCtrl) GetPolicy() Policy{
      return pctl.Policy
}

func (pctl *PolicyCtrl) LoadFromConfig() error {
	config := pctl.configService.GetConfiguration()

	return pctl.LoadFromYaml(config.Policy)
}

// LoadFromYaml load and normalize the result.
//
// For root node, nil means false, should create a node.
// For non-root node, nil means look levels up for that field.
//
// If Days* is inputed but Persist* is nil, set Persist* = false
// If Days* is inputed(!= nil, or value != 0) but Persist* == true, non-sense, error
// If Days* is not inputed(nil), but Persis* == false, error
//
func (pctl *PolicyCtrl) LoadFromYaml(yamlStr string) error {
	pctl.Policy = Policy{}
	if err := yaml.UnmarshalStrict([]byte(yamlStr), &pctl.Policy); err != nil {
		return err
	}

	if err := pctl.checkAndNorm(); err != nil {
		return err
	}

	return nil
}

func (pctl *PolicyCtrl) mandatoryCheck(s *Settings) error {
	if s == nil {
		return nil
	}

	if s.DaysOfPrune != nil {
		if s.DaysOfPrune.Value == 0 {
			return fmt.Errorf("%w field:days_of_prune", ErrRequired)
		}
		if s.PersistNormal != nil && s.PersistNormal.Value {
			return fmt.Errorf("%w field:days_of_prune, persist_normal", ErrConfilct)
		}
	} else {
		if s.PersistNormal != nil && !s.PersistNormal.Value {
			return fmt.Errorf("%w field:days_of_prune, persist_normal", ErrConfilct)
		}
	}

	if s.DaysOfDeleted != nil {
		if s.DaysOfDeleted.Value == 0 {
			return fmt.Errorf("%w field:days_of_deleted", ErrRequired)
		}
		if s.PersistDeleted != nil && s.PersistDeleted.Value {
			return fmt.Errorf("%w field:days_of_deleted, persist_deleted", ErrConfilct)
		}
	} else {
		if s.PersistDeleted != nil && !s.PersistDeleted.Value {
			return fmt.Errorf("%w field:days_of_deleted, persist_deleted", ErrConfilct)
		}
	}

	return nil
}

func (pctl *PolicyCtrl) fillBlankSettings(s *Settings) {
	if s == nil {
		return
	}
	if s.DaysOfPrune != nil && s.DaysOfPrune.Value != 0 &&
		s.PersistNormal == nil {
		s.PersistNormal = &SWITCH_FALSE
	}

	if s.DaysOfDeleted != nil && s.DaysOfDeleted.Value != 0 &&
		s.PersistDeleted == nil {
		s.PersistDeleted = &SWITCH_FALSE
	}
}

// checkAndNorm is just check the input values
// no check is for the expaned value.
// expand values should be checked again
func (pctl *PolicyCtrl) checkAndNorm() error {

	pctl.fillBlankSettings(pctl.Policy.Settings)

	//root node default value
	//no defualt value for Days*, because Persist* = true
	root := pctl.Policy.Settings

	if root == nil {
		pctl.Policy.Settings = &Settings{}
		root = pctl.Policy.Settings
	}

	if root.PersistNormal == nil {
		root.PersistNormal = &SWITCH_TRUE
	}

	if root.PersistDeleted == nil {
		root.PersistDeleted = &SWITCH_TRUE
	}

	if root.OnlyThreadPruned == nil {
		root.OnlyThreadPruned = &SWITCH_FALSE
	}

	if err := pctl.mandatoryCheck(pctl.Policy.Settings); err != nil {
		return fmt.Errorf("%w root", err)
	}

	if pctl.Policy.Teams != nil {
		for tm, s := range pctl.Policy.Teams {
			var team *model.Team
			var err error

			team, err = pctl.pluginAPI.Team.GetByName(tm)
			if err != nil {
				if err == pluginapi.ErrNotFound {
					return fmt.Errorf("%w team:%v", ErrNoTeam, tm)
				}
				return fmt.Errorf("unknown error:%w team:%v", err, tm)
			}

			pctl.fillBlankSettings(s.Settings)

			if err := pctl.mandatoryCheck(s.Settings); err != nil {
				return fmt.Errorf("%w team:%v", err, tm)
			}

			if s.Channels != nil {
				for ch, ss := range s.Channels {
					if _, err := pctl.pluginAPI.Channel.GetByName(team.Id, ch, true); err != nil {
						if err == pluginapi.ErrNotFound {
							return fmt.Errorf("%w channel:%v", ErrNoChannel, ch)
						}
						return fmt.Errorf("unknown error:%w channel:%v", err, ch)
					}

					pctl.fillBlankSettings(ss.Settings)

					if err := pctl.mandatoryCheck(ss.Settings); err != nil {
						return fmt.Errorf("%w channel:%v", err, ch)
					}
				}

			}
		}
	}

	if pctl.Policy.Users != nil {

		pctl.fillBlankSettings(pctl.Policy.Users.Settings)

		if err := pctl.mandatoryCheck(pctl.Policy.Users.Settings); err != nil {
			return fmt.Errorf("%w users", err)
		}
		usrs := pctl.Policy.Users.Channels
		if usrs != nil {
			for usr, u := range usrs {
				if _, err := pctl.pluginAPI.User.GetByUsername(usr); err != nil {
					if err == pluginapi.ErrNotFound {
						return fmt.Errorf("%w user:%v", ErrNoUser, usr)
					}
					return fmt.Errorf("unknown error:%w user:%v", err, usr)
				}

				pctl.fillBlankSettings(u.Settings)

				if err := pctl.mandatoryCheck(u.Settings); err != nil {
					return fmt.Errorf("%w user:%v", err, usr)
				}
			}
		}
	}

	return nil
}

func (pctl *PolicyCtrl) returnNormalizedSettings(levels []*Settings, level int) *Settings {

	var settings *Settings

	checkAndSet := func(set func(*Settings, *Settings), src *Settings) {
		if settings == nil {
			settings = &Settings{}
		}

		set(settings, src)
	}

	lookupFieldValue := func(levels []*Settings, level int,
		notnil func(*Settings) bool) *Settings {

		for level > len(levels)-1 || levels[level] == nil || !notnil(levels[level]) {
			level--
		}

		//up to this point, root must be non-nil
		//so the worst case, root should be returned
		return levels[level]

	}

	for _, fns := range []struct {
		notnil func(*Settings) bool
		set    func(*Settings, *Settings)
	}{
		{
			//setting of DaysOfPrune
			func(s *Settings) bool {
				return s.PersistNormal != nil
			},
			func(s *Settings, src *Settings) {
				// if Persit* == false, Days* must has some values
				s.PersistNormal = src.PersistNormal
				s.DaysOfPrune = src.DaysOfPrune
			},
		},
		{
			//setting of DaysOfDeleted
			func(s *Settings) bool {
				return s.PersistDeleted != nil
			},
			func(s *Settings, src *Settings) {
				s.PersistDeleted = src.PersistDeleted
				s.DaysOfDeleted = src.DaysOfDeleted
			},
		},
		{
			//setting of DaysOfDeleted
			func(s *Settings) bool {
				return s.OnlyThreadPruned != nil
			},
			func(s *Settings, src *Settings) {
				s.OnlyThreadPruned = src.OnlyThreadPruned
			},
		},
	} {
		st := lookupFieldValue(levels, level, fns.notnil)
		if st != nil {
			checkAndSet(fns.set, st)
		}
	}

	if settings == nil {
		return nil
	}

	//because root must have both the values, so we don't check nil
	if settings.PersistNormal.Value && settings.PersistDeleted.Value {
		return nil
	}

	return settings
}

// ExpandPolicyAndNormalize expand the policy to individual unit and normalize the result.
// Normalized result fills all the field settings, which means all the teams/channels/user
// which will be pruned should be filled with all fields.
//
// Expandition flow:
// - If current Persist* is not nil and false, use the Day* vale( Day* must be not zero)
// - if current Persist* is not nil and true, set Persist* = true, leave Day* nil
// - if current Perist* is nil, look levels up until the first Perist* is not nil
// --   if Perist* == true, set true, and leave  Day* nil
// --   if Perist* == false, set false, and set Days* as that value( must not be zero)
// --   if all Persit* is true, don't append the result
//
// Note:
// - Result is is a map with Id( not name)
func (pctl *PolicyCtrl) ExpandPolicyAndNormalize() (*ExpanedPolicyWithId, error) {

	ep := ExpanedPolicyWithId{}

	tmsep, err := pctl.expandTeams()
	if err != nil {
		return nil, err
	}

	if tmsep != nil {
		ep.Teams = tmsep
	}

	usrsep, err := pctl.expandUsers()
	if err != nil {
		return nil, err
	}

	if usrsep != nil {
		ep.Users = usrsep
	}
	return &ep, nil
}

func (pctl *PolicyCtrl) expandTeams() (Teams, error) {

	tms, appErr := pctl.pluginAPI.Team.List(func(o *pluginapi.ListTeamsOptions) {
		o.UserID = ""
	})

	if appErr != nil {
		return nil, appErr
	}

	var tmsep Teams

	slevels := []*Settings{pctl.Policy.Settings}

	for _, tm := range tms {

		tep := Team{
			Channels: Channels{},
		}

		var (
			tp      *Team
			ok      bool
			foundTp bool
		)

		if pctl.Policy.Teams != nil {
			if tp, ok = pctl.Policy.Teams[tm.Name]; ok {
				slevels = append(slevels, tp.Settings)
				foundTp = true
			}
		}
		if !foundTp {
			slevels = append(slevels, nil)
		}

		chs, appErr := pctl.pruneStore.GetAllChannelsForTeam(tm.Id)
		if appErr != nil {
			return nil, appErr
		}

		for _, ch := range chs {

			var (
				foundCp bool
			)

			if tp != nil && tp.Channels != nil {
				if cp, ok := tp.Channels[ch.Name]; ok {
					slevels = append(slevels, cp.Settings)
					foundCp = true
				}
			}

			if !foundCp {
				slevels = append(slevels, nil)
			}

			if cep := pctl.returnNormalizedSettings(slevels, 2); cep != nil {
				tep.Channels[ch.Id] = &Channel{cep}
			}

			slevels = slevels[:len(slevels)-1]
		}

		if len(tep.Channels) != 0 {
			if tmsep == nil {
				tmsep = Teams{}
			}
			tmsep[tm.Id] = &tep
		}

		slevels = slevels[:len(slevels)-1]
	}
	return tmsep, nil
}

func (pctl *PolicyCtrl) expandUsers() (*Users, error) {

	users, appErr := pctl.pluginAPI.User.List(&model.UserGetOptions{})
	if appErr != nil {
		return nil, appErr
	}

	usrsep := &Users{
		Channels: Channels{},
	}

	slevels := []*Settings{pctl.Policy.Settings}

	if pctl.Policy.Users != nil && pctl.Policy.Users.Settings != nil {
		slevels = append(slevels, pctl.Policy.Users.Settings)
	} else {
		slevels = append(slevels, nil)
	}

	for _, usr := range users {

		var (
			foundUsp bool
		)

		if pctl.Policy.Users != nil && pctl.Policy.Users.Channels != nil {
			if p, ok := pctl.Policy.Users.Channels[usr.Username]; ok {
				slevels = append(slevels, p.Settings)
				foundUsp = true
			}
		}

		if !foundUsp {
			slevels = append(slevels, nil)
		}

		chs, appErr := pctl.pluginAPI.Channel.ListForTeamForUser("", usr.Id, true)
		if appErr != nil {
			return nil, appErr
		}

		for _, ch := range chs {

			// just consider direct channel
			if ch.Type != model.ChannelTypeDirect {
				continue
			}

			if cep := pctl.returnNormalizedSettings(slevels, 2); cep != nil {
				//because multi-users may share one channel, so there should be only 1 time
				if _, ok := usrsep.Channels[ch.Id]; ok {
					continue
				}
				usrsep.Channels[ch.Id] = &Channel{cep}
			}

		}

		slevels = slevels[:len(slevels)-1]
	}

	if len(usrsep.Channels) == 0 {
		usrsep = nil
	}
	return usrsep, nil
}

// type SimplePolicy struct {
// 	period  time.Duration
// 	team    SimpleSpecificPolicy
// 	channel SimpleSpecificPolicy
// 	user    SimpleSpecificPolicy
// }
//
// type SimpleSpecificPolicy map[string]time.Duration

// var policy SimplePolicy
//
// func SetPolicy(p SimplePolicy) {
// 	policy = p
// }
//
// func GetPolicy() SimplePolicy {
// 	return policy
// }
//
// func ConvertFromConfig() {
//
// }
