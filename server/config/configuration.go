package config

import (
// "reflect"
//
// "github.com/pkg/errors"
)

// Configuration captures the plugin's external Configuration as exposed in the Mattermost server
// Configuration, as well as values computed from the Configuration. Any public fields will be
// deserialized from the Mattermost server Configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// Configuration can change at any time, access to the Configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the Configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your Configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type Configuration struct {
        Policy        string
	BotUserID     string
	// AdminLogLevel is "debug", "info", "warn", or "error".
	AdminLogLevel string

	// AdminLogVerbose: set to include full context with admin log messages.
	AdminLogVerbose bool
	// ** The following are NOT stored on the server
	// AdminUserIDs contains a list of user IDs that are allowed
	// to administer plugin functions, even if not Mattermost sysadmins.
	AllowedUserIDs []string
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *Configuration) Clone() *Configuration {
	var clone = *c
	return &clone
}

func (c *Configuration) serialize() map[string]interface{} {
	ret := make(map[string]interface{})
	ret["BotUserID"] = c.BotUserID
	return ret
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
// func (p *Plugin) getConfiguration() *Configuration {
// 	p.configurationLock.RLock()
// 	defer p.configurationLock.RUnlock()
//
// 	if p.configuration == nil {
// 		return &Configuration{}
// 	}
//
// 	return p.configuration
// }

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
// func (p *Plugin) setConfiguration(configuration *Configuration) {
// 	p.configurationLock.Lock()
// 	defer p.configurationLock.Unlock()
//
// 	if configuration != nil && p.configuration == configuration {
// 		// Ignore assignment if the configuration struct is empty. Go will optimize the
// 		// allocation for same to point at the same memory address, breaking the check
// 		// above.
// 		if reflect.ValueOf(*configuration).NumField() == 0 {
// 			return
// 		}
//
// 		panic("setConfiguration called with the existing configuration")
// 	}
//
// 	p.configuration = configuration
// }
