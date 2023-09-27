package main

import (
	"fmt"
	// "sort"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneTest(t *testing.T) {
	e := Setup(t)

	e.CreateClients()
	e.CreateBasicServer()

	r, appErr := e.A.GetPlugins()
	require.Nil(t, appErr)

	fmt.Printf("%#v\n", r)

	var myPlugin *model.PluginInfo
	for _, p := range r.Active {
		if strings.HasPrefix(p.Id, "com.github.ericzzh.mattermost-plugin-prune") {
			myPlugin = p
			break
		}
	}

	require.NotNil(t, myPlugin)

	myPluginAPI := e.A.NewPluginAPI(nil, &myPlugin.Manifest)

	t.Run("delete_root", func(t *testing.T) {

		myPluginAPI.SavePluginConfig(map[string]interface{}{
			"Policy": fmt.Sprintf(`
                        settings:
                            days_of_prune:
                                value: %.10f
                                  `, 1.0/(24*3600))})

		pubChannel, _, err := e.AdminClient.CreateChannel(&model.Channel{
			DisplayName: "testpublic1",
			Name:        "testpublic1",
			Type:        model.ChannelTypeOpen,
			TeamId:      e.BasicTeam.Id,
		})
		require.NoError(e.T, err)

		_, _, err = e.AdminClient.AddChannelMember(pubChannel.Id, e.RegularUser.Id)
		require.NoError(e.T, err)

		e.RegularUserClient.CreatePost(
			&model.Post{
				ChannelId: pubChannel.Id,
				Message:   "root message",
			})

		cpOld := selectAllPosts(t, e, pubChannel.Id)
		require.GreaterOrEqual(t, len(cpOld.all), 0)

		time.Sleep(time.Second * 1)

		e.AdminClient.ExecuteCommand(pubChannel.Id, "/prune run")

		cpNew := selectAllPosts(t, e, pubChannel.Id)
		assert.Equal(t, 0, len(cpNew.all))
	})
}

type postsMap map[string]*model.Post
type postsMapList map[string][]*model.Post
type collectedPosts struct {
	all              postsMap
	roots            postsMap
	rootsWithThreads postsMapList
	deleted          postsMap
	nonOriginal      postsMap

	system postsMap
	normal postsMap
	others postsMap

	pinned postsMap
}

func selectAllPosts(t *testing.T, e *TestEnvironment, ch string) (cp collectedPosts) {
	t.Helper()

	cp = collectedPosts{
		all:              postsMap{},
		roots:            postsMap{},
		rootsWithThreads: postsMapList{},
		deleted:          postsMap{},
		nonOriginal:      postsMap{},
		system:           postsMap{},
		normal:           postsMap{},
		others:           postsMap{},
	}

	var posts []*model.Post

	err := e.SqlStore.GetMasterX().Select(&posts, `SELECT * FROM Posts`)
	require.Nil(t, err)

	for _, post := range posts {
		cp.all[post.Id] = post

		if post.RootId == "" {
			cp.roots[post.Id] = post
		} else {
			if _, ok := cp.rootsWithThreads[post.RootId]; !ok {
				cp.rootsWithThreads[post.RootId] = []*model.Post{post}
			}
			cp.rootsWithThreads[post.RootId] = append(cp.rootsWithThreads[post.RootId], post)
		}

		if post.DeleteAt != 0 {
			cp.deleted[post.Id] = post
		}

		if post.OriginalId != "" {
			cp.nonOriginal[post.Id] = post
		}

		switch {
		case post.Type == model.PostTypeDefault || strings.HasPrefix(post.Type, model.PostCustomTypePrefix):
			cp.normal[post.Id] = post
		case strings.HasPrefix(post.Type, model.PostSystemMessagePrefix):
			cp.system[post.Id] = post
		default:
			cp.others[post.Id] = post
		}
	}
	return
}

func toList(t *testing.T, pm postsMap) {
	t.Helper()
}
func printPosts(t *testing.T, pm postsMap) {

	t.Helper()

}
