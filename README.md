# Poor Man's Scheduled Messages

[![Build Status](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml/badge.svg)](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml)
[![CodeQL](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/github-code-scanning/codeql)

<div align="center">
  <img src="logo.png" alt="Poor Man's Scheduled Messages logo">
</div>

## Purpose

The paid version of [Mattermost](https://mattermost.com) comes with [scheduled messages](https://docs.mattermost.com/collaborate/schedule-messages.html), but what about us poor schlubs on the free plan??

The Poor Man's Scheduled Messages plugin aims to fill this gap, albeit less elegantly. What you get is a `/schedule` [slash command](https://docs.mattermost.com/collaborate/run-slash-commands.html) that allows you to schedule text-only messages to be posted at a future time to any channel or direct message.

## Installation

Until an official release is available, the latest release tarball can be installed...

via [System Console](https://developers.mattermost.com/integrate/plugins/components/server/hello-world/#install-the-plugin)

...or...

via [mmctl](https://docs.mattermost.com/manage/mmctl-command-line-tool.html#mmctl-plugin-add)

## Usage

With the plugin installed: `/schedule help`

..or view the help [here](assets/help.md)

### Language and time display

Dates, month names, confirmations, list labels, and action buttons follow the user's Mattermost interface language. The clock format follows the user's personal time preference independently of language: `13:10` for 24-hour time or `1:10 PM` for 12-hour time. The `AM` and `PM` markers remain in English for every language.

Current preferences are read when a response is generated, including when listing existing scheduled messages. Dates use the time zone saved with the scheduled message. If settings cannot be loaded, the plugin falls back to the server's default client language (or English) and a 12-hour clock for the unavailable settings.

Error localization is partial: surrounding labels are translated, while some validation reasons (such as message count and size limits), parser details, and external technical errors remain in English. A response can therefore contain both translated and English text. Delivery state and failure category identifiers are technical English values. Command syntax, autocomplete descriptions, and the help document also remain in English.

### Delivery and recovery

Jobs are retained until their outcome is recorded. The timer and **Send** share
an atomic claim, including across plugin instances. Legacy jobs without a state
are treated as pending. User indexes use compare-and-set updates and are repaired
from canonical job records during scheduler scans.

Creation first persists a non-publishable `creating` record, then atomically
reserves index capacity, and only then marks the job pending. List cleanup checks
the canonical record again before removing an index entry. Interrupted creations
are completed during recovery; a creation rejected by the quota is cancelled
without being published. Index repair for already accepted jobs bypasses the
creation quota so a full index cannot hide an existing job.

The list shows the delivery state, attempt count, failure category, and next
check. A processing claim expires after two minutes. A worker that resumes with
an obsolete claim cannot start a publication or overwrite another worker's result.
Cancellation can win before publication starts; once a request may be in flight,
cancellation is refused rather than claiming the message cannot appear.

An explicit rate-limit rejection (HTTP 429) is retried up to five total attempts,
with delays of 1, 2, 4, and 8 minutes. Other explicit HTTP 4xx rejections, except
timeouts, leave a failed job for inspection, cancellation, or an explicit **Send**
to start another bounded batch. Successful and cancelled jobs become receipts,
with message text removed, and expire after seven days.

Timeouts, transport errors, HTTP 5xx errors, missing publication receipts, and
crashes after publication begins have an **uncertain** outcome. These jobs stay
stored and are checked for an existing post using a persistent job marker.
Recovery runs after an expired claim, then every five minutes; **Check delivery**
also performs reconciliation without re-publishing. If the post is found, the job
is marked sent. If not, the job remains uncertain and its original text remains
available for manual investigation. A lookup examines at most 100,000 recent
channel posts per check. It never interprets an absent or inaccessible post as
permission to publish again.

This deliberately avoids promising exactly-once delivery from an API without a
durable idempotency key. Mattermost rejects preassigned IDs for ordinary posts;
its pending-post deduplication is cache-based. An unknown outcome therefore
requires reconciliation rather than an automatic retry that could create a
duplicate. See the upstream [post store](https://github.com/mattermost/mattermost/blob/master/server/channels/store/sqlstore/post_store.go)
and [post creation](https://github.com/mattermost/mattermost/blob/master/server/channels/app/post.go).

## Caveats

You get what you pay for, so...

1. **No attachments:** You can't attach anything to a scheduled message *(slash commands don't pass attachment data as far as I can tell)*.
2. **Send order:** Messages scheduled for the same channel or direct message at the exact same time are *NOT* guaranteed to be posted in the order they were scheduled.
3. **Message limits:**
   * 1000 scheduled messages per user
   * 50KB per message *(max message length in Mattermost interface is currently about 16KB, so shouldn't be a problem)*.
4. **High performance? Who knows:**
   * Messages are managed via Mattermost's internal key/value store.
   * A scheduler cycles through all scheduled messages once per minute, sending those that are due.
   * If you don't exceed the 'official' free plan limit of fifty users, and your users aren't all scheduling hundreds of messages, it will *probably* be fine.
