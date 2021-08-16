import {Store, Action} from 'redux';

import {GlobalState} from 'mattermost-redux/types/store';

import manifest from './manifest';

// eslint-disable-next-line import/no-unresolved
import {PluginRegistry} from './types/mattermost-webapp';
import TeamPolicy from 'components/TeamPolicy';
import ChannelPolicy from 'components/ChannelPolicy';
import UserPolicy from 'components/UserPolicy';

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-empty-function
    public async initialize(registry: PluginRegistry, store: Store<GlobalState, Action<Record<string, unknown>>>) {
        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/
        registry.registerAdminConsoleCustomSetting('TeamPolicy', TeamPolicy, {showTitle: true});
        registry.registerAdminConsoleCustomSetting('ChannelPolicy', ChannelPolicy, {showTitle: true});
        registry.registerAdminConsoleCustomSetting('UserPolicy', UserPolicy, {showTitle: true});
    }
}

declare global {
    interface Window {
        registerPlugin(id: string, plugin: Plugin): void
    }
}

window.registerPlugin(manifest.id, new Plugin());
