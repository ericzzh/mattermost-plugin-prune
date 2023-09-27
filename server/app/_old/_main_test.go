package app

import (
	// "flag"
	"fmt"
	"os"
	"testing"

	"github.com/mattermost/mattermost-server/v6/api4"
	// "github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/testlib"
)

var replicaFlag bool

// export MM_SERVER_PATH=~/go/src/mattermost-server
// export MM_SQLSETTINGS_DRIVERNAME=mysql
// source ./set_env.sh && go test -v

var mainHelper *testlib.MainHelper

func TestMain(m *testing.M) {

        //Changint working directory
	wd, err := os.Getwd()

	if err != nil {
             fmt.Println(err)
             return
	}
	srvPath := os.Getenv("MM_SERVER_PATH")
	if srvPath == "" {
             fmt.Println("Can't find MM_SERVER_PATH.")
             return
	}
	err = os.Chdir(srvPath)
	if err != nil {
             fmt.Println(err)
             return
	}
	defer os.Chdir(wd)

	fmt.Printf("Changed to working dir: %s\n", srvPath)

        //main test begin
	var options = testlib.HelperOptions{
		EnableStore:     true,
		EnableResources: true,
		WithReadReplica: false,
	}

	// mlog.DisableZap()

        mainHelper = testlib.NewMainHelperWithOptions(&options)
        api4.SetMainHelper(mainHelper)
	defer mainHelper.Close()

	mainHelper.Main(m)
}

// import (
// 	"testing"
//         // "fmt"
// 
// 	"github.com/mattermost/mattermost-server/v5/shared/mlog"
// 	// "github.com/mattermost/mattermost-server/v5/store/sqlstore"
// 	"github.com/mattermost/mattermost-server/v5/testlib"
// 	// "github.com/mattermost/mattermost-server/v5/imports/retention"
// )
// 
// var mainHelper *testlib.MainHelper
// 
// func TestMain(m *testing.M) {
// 	mlog.DisableZap()
// 	mainHelper = testlib.NewMainHelperWithOptions(nil)
// 	defer mainHelper.Close()
// 
//         initStores()
//         // fmt.Println("initStores OK.")
// 
// 	mainHelper.Main(m)
//         tearDownStores()
// }
