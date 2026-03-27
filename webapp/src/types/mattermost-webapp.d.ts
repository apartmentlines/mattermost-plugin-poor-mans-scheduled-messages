import type * as React from 'react';
import type {Store} from 'redux';

export type UniqueIdentifier = string;
type ReactResolvable = React.ComponentType<any>;

export interface PluginRegistry {
    registerPostEditorActionComponent(
        ...args: [ReactResolvable] | [{ component: ReactResolvable }]
    ): UniqueIdentifier;
}

export interface WebappGlobal {
    registerPlugin: (pluginId: string, plugin: { initialize: (registry: PluginRegistry, store: Store) => void }) => void;
}
