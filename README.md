# Poor Man's Scheduled Messages

[![Build Status](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml/badge.svg)](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml)

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

## Caveats

You get what you pay for, so...

1. **No attachments:** Slash commands don't pass attachment data as far as I can tell, so you can't attach anything to a scheduled message.
2. **Message limits:** There is a limit of 1000 scheduled messages per user.
3. **High performance? Who knows:** Messages are managed via Mattermost's internal key/value store, and a scheduler cycles through all scheduled messages once per minute, sending those that are due. I doubt this would perform well on an installation with a ton of users.
