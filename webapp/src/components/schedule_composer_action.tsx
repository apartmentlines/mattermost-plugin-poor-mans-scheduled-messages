import React, {useCallback, useEffect, useMemo, useState} from 'react';
import type {CSSProperties} from 'react';
import ReactDOM from 'react-dom';

import {Client4} from 'mattermost-redux/client';

import type {UILang} from '../i18n/ui';
import {scheduleUiStrings} from '../i18n/ui';
import manifest from '../manifest';

type PostDraft = {
    message: string;
    channelId: string;
    rootId: string;
};

type Props = {
    draft: PostDraft;

    /** Derived in bound wrapper from Mattermost store + html lang (not react-redux context). */
    uiLang: UILang;

    /** Mattermost passes a single string (new message body), not a full draft. */
    updateText: (message: string) => void;
};

function pad2(n: number): string {
    return String(n).padStart(2, '0');
}

/** Next full hour as local date + time strings for <input type="date"> and <input type="time">. */
function defaultDateAndTime(): {date: string; time: string} {
    const d = new Date(Date.now() + (60 * 60 * 1000));
    d.setMinutes(0, 0, 0);
    return {
        date: `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`,
        time: `${pad2(d.getHours())}:${pad2(d.getMinutes())}`,
    };
}

function localDateTimeToDate(dateStr: string, timeStr: string): Date {
    return new Date(`${dateStr}T${timeStr}`);
}

const backdropStyle: CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0, 0, 0, 0.45)',
    zIndex: 2000,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
};

const panelStyle: CSSProperties = {
    background: 'var(--center-channel-bg, #fff)',
    color: 'var(--center-channel-color, #333)',
    padding: 0,
    borderRadius: '8px',
    width: 'min(520px, 92vw)',
    maxWidth: '92vw',
    boxShadow: '0 12px 40px rgba(0,0,0,0.25)',
    border: '1px solid rgba(0,0,0,0.08)',
    overflow: 'hidden',
};

const modalHeaderStyle: CSSProperties = {
    padding: '20px 24px 12px',
    borderBottom: '1px solid rgba(var(--center-channel-color-rgb), 0.12)',
};

const modalTitleStyle: CSSProperties = {
    margin: 0,
    fontSize: '22px',
    fontWeight: 700,
    lineHeight: 1.3,
};

const modalTimeZoneStyle: CSSProperties = {
    margin: '8px 0 0',
    fontSize: '13px',
    opacity: 0.75,
    lineHeight: 1.4,
};

const modalBodyStyle: CSSProperties = {
    padding: '20px 24px',
};

const dateTimePickerRowStyle: CSSProperties = {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '12px',
    alignItems: 'stretch',
};

const pickerHalfStyle: CSSProperties = {
    flex: '1 1 200px',
    minWidth: 0,
};

const inputWithLeftIconStyle: CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    padding: '0 10px 0 12px',
    minHeight: '40px',
    borderRadius: '4px',
    border: '1px solid rgba(var(--center-channel-color-rgb), 0.16)',
    background: 'var(--center-channel-bg, #fff)',
    boxSizing: 'border-box',
};

const pickerInputStyle: CSSProperties = {
    flex: 1,
    minWidth: 0,
    border: 'none',
    outline: 'none',
    background: 'transparent',
    color: 'inherit',
    fontSize: '15px',
    lineHeight: '1.25',
    padding: '10px 0',
};

const iconWrapStyle: CSSProperties = {
    flexShrink: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    opacity: 0.65,
    color: 'inherit',
};

const modalFooterStyle: CSSProperties = {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: '12px',
    padding: '16px 24px 20px',
    borderTop: '1px solid rgba(var(--center-channel-color-rgb), 0.12)',
};

const errorStyle: CSSProperties = {
    color: 'var(--error-text, #c0392b)',
    fontSize: '13px',
    marginTop: '12px',
    lineHeight: 1.4,
};

/** Mirrors Mattermost `IconContainer` in formatting_bar/formatting_icon.tsx (no styled-components in plugin). */
function scheduleFormatBarButtonStyle(hover: boolean): CSSProperties {
    const bg = hover ? 'rgba(var(--center-channel-color-rgb), 0.08)' : 'transparent';
    let fg = 'rgba(var(--center-channel-color-rgb), var(--icon-opacity, 0.56))';
    if (hover) {
        fg = 'rgba(var(--center-channel-color-rgb), var(--icon-opacity-hover, 0.72))';
    }
    return {
        display: 'flex',
        minWidth: 32,
        height: 32,
        alignItems: 'center',
        justifyContent: 'center',
        border: 'none',
        background: bg,
        padding: '0 7px',
        borderRadius: 4,
        color: fg,
        cursor: 'pointer',
        boxSizing: 'border-box',
        fill: 'currentColor',
    };
}

function ScheduleClockIcon(): React.ReactElement {
    return (
        <svg
            xmlns='http://www.w3.org/2000/svg'
            version='1.1'
            width='18'
            height='18'
            fill='currentColor'
            viewBox='0 0 24 24'
            aria-hidden='true'
        >
            <path d={'M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z'}/>
        </svg>
    );
}

function mattermostReactBootstrap(): typeof ReactBootstrap | null {
    if (typeof window === 'undefined') {
        return null;
    }
    const w = window as Window & {ReactBootstrap?: typeof ReactBootstrap};
    return w.ReactBootstrap ?? null;
}

function ScheduleCalendarIcon(): React.ReactElement {
    return (
        <svg
            xmlns='http://www.w3.org/2000/svg'
            width='18'
            height='18'
            viewBox='0 0 24 24'
            fill='currentColor'
            aria-hidden='true'
        >
            <path d={'M19 4h-1V2h-2v2H8V2H6v2H5c-1.11 0-1.99.9-1.99 2L3 20a2 2 0 002 2h14c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 16H5V10h14v10zm0-12H5V6h14v2z'}/>
        </svg>
    );
}

export default function ScheduleComposerAction({draft, uiLang, updateText}: Props) {
    const ui = useMemo(() => scheduleUiStrings(uiLang), [uiLang]);
    const [barBtnHover, setBarBtnHover] = useState(false);
    const [open, setOpen] = useState(false);
    const [{date: dateLocal, time: timeLocal}, setDateTime] = useState(defaultDateAndTime);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const messageStr =
        draft?.message != null && typeof draft.message === 'string' ? draft.message : '';

    const openModal = useCallback(() => {
        setDateTime(defaultDateAndTime());
        setError(null);
        setOpen(true);
    }, []);

    const closeModal = useCallback(() => {
        if (!busy) {
            setOpen(false);
        }
    }, [busy]);

    useEffect(() => {
        if (!open) {
            return undefined;
        }
        const onKey = (ev: KeyboardEvent) => {
            if (ev.key === 'Escape') {
                closeModal();
            }
        };
        document.addEventListener('keydown', onKey);
        return () => document.removeEventListener('keydown', onKey);
    }, [open, closeModal]);

    const submit = useCallback(async () => {
        const text = messageStr.trim();
        if (!text) {
            setError(ui.enterMessageFirst);
            return;
        }
        const t = localDateTimeToDate(dateLocal, timeLocal);
        if (Number.isNaN(t.getTime())) {
            setError(ui.pickValidDateTime);
            return;
        }
        setBusy(true);
        setError(null);
        try {
            const url = `/plugins/${manifest.id}/api/v1/schedule`;
            const res = await fetch(url, Client4.getOptions({
                method: 'POST',
                body: JSON.stringify({
                    channel_id: draft.channelId,
                    root_id: draft.rootId || '',
                    message: text,
                    post_at: t.toISOString(),
                }),
            }));
            const bodyText = await res.text();
            if (!res.ok) {
                setError(bodyText || ui.requestFailed(res.status));
                return;
            }
            updateText('');
            setTimeout(() => {
                updateText('');
            }, 0);
            setOpen(false);
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : ui.networkError);
        } finally {
            setBusy(false);
        }
    }, [draft.channelId, draft.rootId, dateLocal, timeLocal, messageStr, updateText, ui]);

    const timeZoneLine = useMemo(() => {
        if (!open) {
            return '';
        }
        try {
            const iana = new Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
            return ui.timeZoneLine(iana);
        } catch {
            return ui.timeZoneLine('UTC');
        }
    }, [open, ui]);

    const modal = useMemo(() => {
        if (!open) {
            return null;
        }
        return ReactDOM.createPortal(
            (
                <div
                    role='presentation'
                    style={backdropStyle}
                    onClick={closeModal}
                >
                    <div
                        role='dialog'
                        aria-modal='true'
                        aria-labelledby='schedule-send-modal-title'
                        aria-describedby='schedule-send-timezone-label'
                        style={panelStyle}
                        onClick={(e) => e.stopPropagation()}
                    >
                        <div style={modalHeaderStyle}>
                            <h1
                                id='schedule-send-modal-title'
                                style={modalTitleStyle}
                            >
                                {ui.modalTitle}
                            </h1>
                            <p
                                id='schedule-send-timezone-label'
                                style={modalTimeZoneStyle}
                            >
                                {timeZoneLine}
                            </p>
                        </div>
                        <div style={modalBodyStyle}>
                            <div
                                id='date_time_picker_plugin'
                                style={dateTimePickerRowStyle}
                                data-testid='schedule-message-date-time-row'
                            >
                                <div style={pickerHalfStyle}>
                                    <div style={inputWithLeftIconStyle}>
                                        <span style={iconWrapStyle}>
                                            <ScheduleCalendarIcon/>
                                        </span>
                                        <input
                                            id='schedule-send-date'
                                            type='date'
                                            value={dateLocal}
                                            onChange={(e) => setDateTime((prev) => ({
                                                ...prev,
                                                date: e.target.value,
                                            }))}
                                            disabled={busy}
                                            aria-label={`${ui.ariaScheduleMessage} Date`}
                                            aria-required={true}
                                            data-testid='schedule-message-date-input'
                                            style={pickerInputStyle}
                                        />
                                    </div>
                                </div>
                                <div style={pickerHalfStyle}>
                                    <div style={inputWithLeftIconStyle}>
                                        <span style={iconWrapStyle}>
                                            <ScheduleClockIcon/>
                                        </span>
                                        <input
                                            id='schedule-send-time'
                                            type='time'
                                            value={timeLocal}
                                            onChange={(e) => setDateTime((prev) => ({
                                                ...prev,
                                                time: e.target.value,
                                            }))}
                                            disabled={busy}
                                            aria-label={`${ui.ariaScheduleMessage} Time`}
                                            aria-required={true}
                                            data-testid='schedule-message-time-input'
                                            style={pickerInputStyle}
                                        />
                                    </div>
                                </div>
                            </div>
                            {error && (
                                <div style={errorStyle}>
                                    {error}
                                </div>
                            )}
                        </div>
                        <div style={modalFooterStyle}>
                            <button
                                type='button'
                                className='btn btn-tertiary'
                                aria-label={ui.cancel}
                                onClick={closeModal}
                                disabled={busy}
                            >
                                {ui.cancel}
                            </button>
                            <button
                                type='button'
                                className='btn btn-primary'
                                aria-label={ui.scheduleMessage}
                                data-testid='schedule-message-submit'
                                onClick={() => {
                                    submit().catch(() => null);
                                }}
                                disabled={busy}
                            >
                                {busy ? '…' : ui.scheduleMessage}
                            </button>
                        </div>
                    </div>
                </div>
            ),
            document.body,
        );
    }, [open, dateLocal, timeLocal, error, busy, closeModal, submit, timeZoneLine, ui]);

    const scheduleButton = (
        <button
            type='button'
            className='style--none control'
            data-testid='schedule-message-plugin-format-bar'
            aria-label={ui.ariaScheduleMessage}
            onClick={openModal}
            onMouseEnter={() => setBarBtnHover(true)}
            onMouseLeave={() => setBarBtnHover(false)}
            style={scheduleFormatBarButtonStyle(barBtnHover)}
        >
            <ScheduleClockIcon/>
        </button>
    );

    const rb = mattermostReactBootstrap();
    const RBOverlay = rb?.OverlayTrigger;
    const RBTooltip = rb?.Tooltip;
    const hasRb = RBOverlay && RBTooltip;

    return (
        <>
            {hasRb ? (
                <RBOverlay
                    placement='top'
                    delay={{show: 400, hide: 200}}
                    overlay={(
                        <RBTooltip id='schedule-message-plugin-tooltip'>
                            {ui.tooltipBase}
                        </RBTooltip>
                    )}
                >
                    {scheduleButton}
                </RBOverlay>
            ) : (
                React.cloneElement(scheduleButton, {title: ui.tooltipBase})
            )}
            {modal}
        </>
    );
}
