/** UI strings for the scheduling modal and format-bar control (browser / Mattermost language). */

export type UILang = 'en' | 'uk';

/** Maps Mattermost account locale (e.g. from getCurrentUserLocale) to bundled UI language. */
export function mattermostLocaleToUiLang(mmLocale: string | undefined | null): UILang {
    const normalized = (mmLocale ?? '').trim().toLowerCase().replace(/_/g, '-');

    // Ukrainian (ISO 639-1 `uk`; Mattermost uses e.g. uk, uk-UA).
    if (normalized === 'uk' || normalized.startsWith('uk-')) {
        return 'uk';
    }
    return 'en';
}

/**
 * Merge Redux profile locale with visible UI signals. Plugins often lack reliable react-redux context;
 * Mattermost also syncs account language to the html[lang] attribute (see MM-48048).
 */
export function resolvePluginUiLang(reduxUserLocale: string | undefined | null): UILang {
    if (mattermostLocaleToUiLang(reduxUserLocale) === 'uk') {
        return 'uk';
    }
    if (typeof document !== 'undefined') {
        const htmlLang = document.documentElement.getAttribute('lang');
        if (mattermostLocaleToUiLang(htmlLang) === 'uk') {
            return 'uk';
        }
    }
    if (typeof navigator !== 'undefined' && mattermostLocaleToUiLang(navigator.language) === 'uk') {
        return 'uk';
    }
    return 'en';
}

export type ScheduleUiStrings = {
    modalTitle: string;
    timeZoneLine: (iana: string) => string;
    ariaScheduleMessage: string;
    cancel: string;
    scheduleMessage: string;
    enterMessageFirst: string;
    pickValidDateTime: string;
    networkError: string;
    requestFailed: (status: number) => string;
    tooltipBase: string;
};

const en: ScheduleUiStrings = {
    modalTitle: 'Schedule message',
    timeZoneLine: (iana) => `Time zone: ${iana}`,
    ariaScheduleMessage: 'Schedule message',
    cancel: 'Cancel',
    scheduleMessage: 'Schedule Message',
    enterMessageFirst: 'Enter a message in the composer first.',
    pickValidDateTime: 'Pick a valid date and time.',
    networkError: 'Network error',
    requestFailed: (status) => `Request failed (${status})`,
    tooltipBase: 'Schedule message',
};

const uk: ScheduleUiStrings = {
    modalTitle: 'Запланувати повідомлення',
    timeZoneLine: (iana) => `Часовий пояс: ${iana}`,
    ariaScheduleMessage: 'Запланувати повідомлення',
    cancel: 'Скасувати',
    scheduleMessage: 'Запланувати',
    enterMessageFirst: 'Спочатку введіть текст у полі повідомлення.',
    pickValidDateTime: 'Оберіть коректну дату й час.',
    networkError: 'Помилка мережі',
    requestFailed: (status) => `Запит не вдався (${status})`,
    tooltipBase: 'Запланувати повідомлення',
};

const byLang: Record<UILang, ScheduleUiStrings> = {en, uk};

export function scheduleUiStrings(lang: UILang): ScheduleUiStrings {
    return byLang[lang] ?? en;
}
