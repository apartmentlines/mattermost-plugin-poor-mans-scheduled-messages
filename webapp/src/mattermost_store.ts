// Holds the Redux store reference provided by Mattermost in initialize(registry, store).
import type {Store} from 'redux';

let mattermostStore: Store | null = null;

export function setMattermostStore(store: Store): void {
    mattermostStore = store;
}

export function getMattermostStore(): Store | null {
    return mattermostStore;
}
