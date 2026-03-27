// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Store} from 'redux';

import manifest from './manifest';
import {setMattermostStore} from './mattermost_store';
import ScheduleComposerActionBound from './schedule_composer_bound';
import type {PluginRegistry} from './types/mattermost-webapp';

class Plugin {
    public initialize(registry: PluginRegistry, store: Store): void {
        setMattermostStore(store);
        registry.registerPostEditorActionComponent(ScheduleComposerActionBound);
    }
}

declare global {
    interface Window {
        registerPlugin: (pluginId: string, plugin: Plugin) => void;
    }
}

window.registerPlugin(manifest.id, new Plugin());

export default Plugin;
