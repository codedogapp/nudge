# Setup

How to install nudge on a Mac, and what it needs before `nudge serve` can run: its settings, and `tailscale serve` to publish the phone door on the tailnet. [The plan](plan.md) says why (§3, decisions 7, 8, 18 and 20). The last section adds nudge to an agent.

## Install

mise installs nudge from its GitHub releases:

```sh
mise use -g github:codedogapp/nudge
```

mise checks the release's digest and its build provenance attestation before it installs it. When it looks for the newest release, it skips those less than 24 hours old (its `minimum_release_age` setting): add `--minimum-release-age 0s` to take a new one at once. `mise upgrade` takes the newest release later.

The binary is then `~/.local/share/mise/installs/github-codedogapp-nudge/latest/nudge`, a path that an upgrade keeps. Agents start it from there ([Agents](#agents)).

## Settings

`nudge serve` reads its settings from the environment. When it runs from the source, they live in `mise.local.toml`, at the repo's root: git ignores it, and mise puts its `[env]` into every task's environment.

```toml
[env]
NUDGE_LOGIN = "user@example.com"
NUDGE_HOST = "machine.tailnet.ts.net"
```

| Variable | Holds | When unset |
| --- | --- | --- |
| `NUDGE_LOGIN` | The one Tailscale login let in | `nudge serve` exits 1 |
| `NUDGE_HOST` | The Mac's ts.net name, which `tailscale serve` publishes, without the trailing dot | `nudge serve` exits 1 |
| `NUDGE_LISTEN` | The phone door's address. Keep it on loopback: anyone who reaches it can send the headers that the lock checks | `127.0.0.1:8700` |
| `NUDGE_SECURITY` | The `security` tool that keeps the VAPID key. Only the tests change it, to point at a stand-in | `/usr/bin/security` |
| `NUDGE_SOCKET` | The agent door's socket, for `nudge serve`, `nudge ask` and `nudge mcp` | `~/Library/Application Support/nudge/agent.sock` |
| `NUDGE_SUBSCRIPTIONS` | The file that keeps the devices' push subscriptions | `~/Library/Application Support/nudge/subscriptions.json` |

Tailscale knows the login and the name:

```sh
tailscale status --json | jq -r '.User[(.Self.UserID | tostring)].LoginName'
tailscale status --json | jq -r '.Self.DNSName | rtrimstr(".")'
```

## Tailscale

Publish the phone door on port 8443 of the Mac's tailnet name:

```sh
tailscale serve --bg --https=8443 8700
```

The page is then at `https://<machine>.<tailnet>.ts.net:8443`. While `nudge serve` isn't running, it answers 502 Bad Gateway. `tailscale serve status` must show the tailnet only, never Funnel. To stop publishing it:

```sh
tailscale serve --https=8443 off
```

## Run

```sh
mise run serve
```

`nudge serve` runs until Ctrl-C. It logs each refused request to stderr, with the login and the host it came with. In another shell, ask a question:

```sh
go run . ask --option Yes --option No "Merge the pull request?"
```

Answer it on the page: tap an option, send a reply, or type a reply and then tap an option to send both. `nudge ask` prints the answer and exits 0.

## The VAPID key

`nudge serve` signs its Web Pushes with a VAPID key. On its first start, it makes one and adds it to the login Keychain as service `nudge`, account `vapid-private-key`; later starts read it from there ([the plan](plan.md), decision 11). To print it:

```sh
security find-generic-password -s nudge -a vapid-private-key -w
```

`nudge serve` never replaces the key. For a new one, delete it and start `nudge serve` again; every device must then turn its notifications on again:

```sh
security delete-generic-password -s nudge -a vapid-private-key
```

When `security` fails, for example on a locked Keychain that can't ask for its password, or the stored key isn't valid, `nudge serve` exits 1 and says why.

## The subscriptions

Each device that turns its notifications on adds its push subscription to `~/Library/Application Support/nudge/subscriptions.json`, or to the file that `NUDGE_SUBSCRIPTIONS` names: a JSON list that only the user can read. To see it:

```sh
cat ~/Library/Application\ Support/nudge/subscriptions.json
```

`nudge serve` keeps the list in memory and writes the whole file on each change, so stop it before you edit or delete the file. To forget every device, stop `nudge serve`, delete the file and start it again. A device that opens the page again sends its subscription again, so it comes back ([Notifications](#notifications)).

When the file can't be read, isn't a JSON list, or holds a subscription that isn't valid, `nudge serve` exits 1 and names the file ([the plan](plan.md), decision 13). Fix the file, or delete it to forget every device.

## The Home Screen

On the iPhone, open the page in Safari, tap Share, then Add to Home Screen. nudge then opens as an app of its own, with its icon and without Safari's bars. Web Push needs it: only a Home Screen app can get notifications on the iPhone ([the plan](plan.md), fact 4).

## Notifications

Open the page, tap Turn on notifications at the top right, and allow them when the browser asks. The header then says Notifications on. A browser's first time can take half a minute. On the iPhone, do it in the Home Screen app: a Safari tab says Notifications need the Home Screen app. Brave needs Use Google services for push messaging, in its privacy settings ([the plan](plan.md), fact 4).

A tap on a notification opens the page, or loads it again where it's already open; a reply half-typed there is lost ([the plan](plan.md), decision 15).

Each new question sends one notification to every device that has turned them on. A notification can come late or not at all, so the page always lists every pending question. `nudge serve`'s log says `forgot a device` when a push service says a device has unsubscribed, and `push failed` or `push refused` when a push didn't go out ([the plan](plan.md), decision 16). A device that has just turned notifications on can be forgotten by mistake ([the plan](plan.md), risk 8): opening the page again brings it back.

Each visit sends the device's subscription to `nudge serve` again ([the plan](plan.md), decision 14). There is no button to turn notifications off: turn them off in the device's notification settings, or block them in the browser's settings for the page.

## Agents

An agent asks through `nudge mcp`, an MCP server with one tool, `ask` ([the plan](plan.md), decision 17). The agent starts the server itself, from the binary that mise installed ([Install](#install)). Add it to Copilot CLI, with a timeout of a day:

```sh
copilot mcp add --timeout 86400000 nudge -- ~/.local/share/mise/installs/github-codedogapp-nudge/latest/nudge mcp
```

Not through mise's shim, `~/.local/share/mise/shims/nudge`: Copilot CLI starts the server in the session's folder, and the shim refuses to run in a folder whose `mise.toml` mise doesn't trust ([the plan](plan.md), fact 9).

Copilot CLI ends a tool call that runs longer than the timeout, and the question goes away with it. Without `--timeout`, Copilot CLI 1.0.102 ended a question after three minutes (risk 9). `copilot mcp add` keeps the server in `~/.copilot/mcp-config.json`, and `copilot mcp get nudge` shows it.

Copilot CLI starts `nudge mcp` in the session's folder, so a question's project is that folder's name. `nudge mcp` reaches `nudge serve` through the same socket as `nudge ask`: when `NUDGE_SOCKET` isn't the default, add `--env NUDGE_SOCKET=<path>` before the `--`.

Copilot CLI can ask at the Mac for permission before the agent uses the tool, which defeats the point when you're away. Start it with `--allow-tool 'nudge(ask)'` to let the agent ask without that prompt.
