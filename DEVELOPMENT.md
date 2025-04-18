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

### Unit‑testing, mocks and fakes

We use two strategies for test doubles:

* **Hand‑written fakes** – for very small, deterministic collaborators  (e.g. Clock, IDGenerator). These live directly in the `_test.go` files.

* **gomock‑generated mocks** – for any larger interface in `internal/ports` (PostService, ChannelService, KVStore, …). Generated mocks belong in `adapters/mock`.

#### Generating mocks

1. Regenerate all mocks whenever an interface in `internal/ports` changes:

       make mocks

2. Commit the regenerated `*_mock.go` files so CI and other developers don’t need `mockgen` installed.

Guideline: if an interface has more than two trivial methods, prefer a gomock‑generated mock; otherwise write a hand fake.

### Architecture – Ports & Adapters

The codebase follows a clean “ports and adapters” pattern to isolate all Mattermost‑API calls behind narrow interfaces declared in `internal/ports`.

          ┌────────────────────┐
          │   Mattermost API   │←── adapters/mm (production)
          └─────────▲──────────┘
                    │implements
          ┌─────────┴──────────┐
          │       ports        │←── internal/ports
          └─────────▲──────────┘
                    │imported by
          ┌─────────┴──────────┐
          │ Application code   │ (scheduler, command, store, channel, …)
          └─────────┬──────────┘
                    │mocked by
          ┌─────────▼──────────┐
          │  test adapters     │←── adapters/mock + hand fakes
          └────────────────────┘

Why  
• Single, central list of external capabilities.  
• Application packages depend only on small, purpose‑built interfaces.  
• Production wiring lives in one place (`plugin.go`).  
• Unit tests swap in gomock mocks or hand fakes with zero Mattermost dependencies.

#### Adding a new capability

1. Edit `internal/ports/ports.go` and extend the appropriate interface or add a new one.

2. Regenerate mocks:

       make mocks

3. Implement the method in the production adapter  (usually this means adding a thin wrapper method to the relevant `pluginapi.*Service` value in `plugin.go`).

4. Inject the new port into any package that needs it via constructor parameters.

5. Update tests to set expectations on the new gomock mock or adjust hand fakes.

