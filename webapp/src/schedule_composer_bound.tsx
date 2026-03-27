// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react';

import ScheduleComposerAction from './components/schedule_composer_action';
import {usePluginUiLang} from './hooks/use_plugin_ui_lang';

type PluginDraft = {
    message?: unknown;
    channelId: string;
    rootId: string;
};

type PluginProps = {
    draft: PluginDraft;
    updateText: (message: string) => void;
    getSelectedText: () => unknown;
};

/**
 * Mattermost passes valid updateText(message: string). If an older bundle ever called
 * updateText(non-string), message in Redux could become non-string and break use_priority
 * (specialMentionsInText expects a string). This wrapper always forwards a string.
 */
export default function ScheduleComposerActionBound({
    draft,
    updateText,
    getSelectedText: _getSelectedText,
}: PluginProps) {
    const uiLang = usePluginUiLang();

    const safeUpdateText = useCallback((message: string) => {
        if (typeof updateText !== 'function') {
            return;
        }
        const m = typeof message === 'string' ? message : String(message ?? '');
        updateText(m);
    }, [updateText]);

    const normalizedDraft = {
        channelId: draft.channelId,
        rootId: draft.rootId || '',
        message: typeof draft.message === 'string' ? draft.message : '',
    };

    return (
        <ScheduleComposerAction
            draft={normalizedDraft}
            uiLang={uiLang}
            updateText={safeUpdateText}
        />
    );
}
