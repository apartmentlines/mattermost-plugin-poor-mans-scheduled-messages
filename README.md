# Poor Man's Scheduled Messages

[![Build Status](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml/badge.svg)](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/ci.yml)
[![E2E Status](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/e2e.yml/badge.svg)](https://github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/actions/workflows/e2e.yml)

## Purpose

The paid version of [Mattermost](https://mattermost.com) comes with [scheduled messages](https://docs.mattermost.com/collaborate/schedule-messages.html), but what about us poor schlubs on the free plan??

The Poor Man's Scheduled Messages plugin aims to fill this gap, albeit less elegantly. What you get is a `/schedule` [slash command](https://docs.mattermost.com/collaborate/run-slash-commands.html) that allows you to schedule text-only messages to be posted at a future time to any channel or direct message.

## Usage

With the plugin installed: `/schedule help`

..or view the help [here](assets/help.md)

## Caveats

You get what you pay for, so...

1. **No attachments:** Slash commands don't pass attachment data as far as I can tell, so you can't attach anything to a scheduled message.
2. **Message limits:** There is a limit of 1000 scheduled messages per user.
3. **High performance? Who knows:** Messages are managed via Mattermost's internal key/value store, and a scheduler cycles through all scheduled messages once per minute, sending those that are due. I doubt this would perform well on an installation with a ton of users.

## Installation

Until an official release is available:

1. Clone this repository
2. Make sure you have your Go development environment configuration
3. From the repository root, run `make`
4. Release tarballs will be available in the `dists/` directory
5. Install the tarball via the Mattermost System Console or the `mmctl` CLI tool

## Development

To avoid having to manually install your plugin, build and deploy your plugin using one of the following options. In order for the below options to work, you must first enable plugin uploads via your config.json or API and restart Mattermost.

```json
    "PluginSettings" : {
        ...
        "EnableUploads" : true
    }
```

### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin. Edit your server configuration as follows:

```json
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    },
}
```

and then deploy your plugin:
```
make deploy
```

You may also customize the Unix socket path:
```bash
export MM_LOCALSOCKETPATH=/var/tmp/alternate_local.socket
make deploy
```

### Releasing new versions

The version of a plugin is determined at compile time, automatically populating a `version` field in the [plugin manifest](plugin.json):
* If the current commit matches a tag, the version will match after stripping any leading `v`, e.g. `1.3.1`.
* Otherwise, the version will combine the nearest tag with `git rev-parse --short HEAD`, e.g. `1.3.1+d06e53e1`.
* If there is no version tag, an empty version will be combined with the short hash, e.g. `0.0.0+76081421`.

To disable this behaviour, manually populate and maintain the `version` field.

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.
