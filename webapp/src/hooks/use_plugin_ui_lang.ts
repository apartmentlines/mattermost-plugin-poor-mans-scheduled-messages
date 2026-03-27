import {useEffect, useState} from 'react';

import type {GlobalState} from '@mattermost/types/store';

import {getCurrentUserLocale} from 'mattermost-redux/selectors/entities/i18n';

import type {UILang} from '../i18n/ui';
import {resolvePluginUiLang} from '../i18n/ui';
import {getMattermostStore} from '../mattermost_store';

function reduxLocaleFromStore(): string {
    const store = getMattermostStore();
    if (!store) {
        return '';
    }
    try {
        return getCurrentUserLocale(store.getState() as GlobalState);
    } catch {
        return '';
    }
}

/**
 * Resolves UI language using the store passed to plugin initialize() (not react-redux context),
 * plus html[lang] and navigator when those indicate Ukrainian.
 */
export function usePluginUiLang(): UILang {
    const [lang, setLang] = useState<UILang>(() => resolvePluginUiLang(reduxLocaleFromStore()));

    useEffect(() => {
        const tick = () => {
            setLang(resolvePluginUiLang(reduxLocaleFromStore()));
        };

        tick();

        const unsubs: Array<() => void> = [];
        const store = getMattermostStore();
        if (store) {
            unsubs.push(store.subscribe(tick));
        }

        if (typeof document !== 'undefined' && typeof MutationObserver !== 'undefined') {
            const obs = new MutationObserver(tick);
            obs.observe(document.documentElement, {attributes: true, attributeFilter: ['lang']});
            unsubs.push(() => obs.disconnect());
        }

        return () => unsubs.forEach((u) => {
            u();
        });
    }, []);

    return lang;
}
