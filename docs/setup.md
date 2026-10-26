# Setup

How to install nudge on a Mac with Homebrew, what it needs before `nudge serve` can run (its settings, and `tailscale serve` to publish the phone door on the tailnet), and how to run it at login. [The plan](plan.md) says why (§3, decisions 7, 8, 23, 24 and 26 to 28). The last two sections add nudge to an agent, and give the agent the skill that sends its questions to the phone.

## Install

Homebrew installs nudge from codedogapp's tap, [`codedogapp/homebrew-tap`](https://github.com/codedogapp/homebrew-tap):

```sh
brew install codedogapp/tap/nudge
```

The formula builds nudge from its tag's source on the Mac, so Homebrew installs Go for the build, or upgrades it.

Give every `brew` command the formula's full name, `codedogapp/tap/nudge`. Homebrew's own cask `nudge` is another app, and a bare `nudge` in `brew install`, `brew upgrade` or `brew info` means that cask. The full name in `brew install` also trusts the formula: Homebrew loads formulae from taps other than its own only once they're trusted ([the plan](plan.md), fact 14).

`nudge` is then on the `PATH`. Homebrew also links it at `$(brew --prefix codedogapp/tap/nudge)/bin/nudge`, a path that stays the same across upgrades; agents start it from there ([Agents](#agents)). To take a new version:

```sh
brew upgrade codedogapp/tap/nudge
```

Then restart the service, which still runs the old one ([Run at login](#run-at-login)).

## Quick setup

On a new Mac, install nudge ([Install](#install)) and run:

```sh
nudge setup
```

It does the steps of [Settings](#settings), [Tailscale](#tailscale) and [Agents](#agents) in one go, and prints a line for each ([the plan](plan.md), decision 33): it writes the settings file from `tailscale status --json`, unless the file exists; it publishes the phone door with `tailscale serve`; and it adds the MCP server and the skill to Copilot CLI, leaving an MCP server that is already there. It is safe to run again. It doesn't start or restart the service: do that with [Run at login](#run-at-login), then do [The Home Screen](#the-home-screen) and [Notifications](#notifications) on the phone. The sections named above say what each step does, and how to do it by hand.

## Settings

`nudge serve` takes each setting from its variable in the environment, else from its key in the settings file, `~/Library/Application Support/nudge/settings.json`, else from its default. An empty value counts as unset. It reads them once, when it starts, so restart it after a change.

| Variable | Key | Holds | When unset |
| --- | --- | --- | --- |
| `NUDGE_LOGIN` | `login` | The one Tailscale login let in | `nudge serve` exits 1 |
| `NUDGE_HOST` | `host` | The Mac's ts.net name, which `tailscale serve` publishes, without the trailing dot | `nudge serve` exits 1 |
| `NUDGE_LISTEN` | `listen` | The phone door's address. Keep it on loopback: anyone who reaches it can send the headers that the lock checks | `127.0.0.1:8700` |
| `NUDGE_SECURITY` | | The `security` tool that keeps the VAPID key. Only the tests change it, to point at a stand-in | `/usr/bin/security` |
| `NUDGE_SOCKET` | | The agent door's socket, for `nudge serve`, `nudge ask` and `nudge mcp`. It has no key, because `nudge ask` and `nudge mcp` read only the environment | `~/Library/Application Support/nudge/agent.sock` |
| `NUDGE_VAPID_KEY_FILE` | | A file for the VAPID key, for a host with no Keychain, such as a container. `nudge serve` makes the key there on its first start, with mode 0600, or reads the one that is there ([The VAPID key](#the-vapid-key)) | The login Keychain |
| `NUDGE_SERVER` | | For `nudge ask` and `nudge mcp` on another machine: the https URL of the machine's `nudge serve` on the tailnet, such as `https://machine.tailnet.ts.net:8443`. When it is set, they use it in place of the socket, and the Tailscale login of the machine must be the one that `nudge serve` lets in ([the plan](plan.md), decision 35) | The socket |
| `NUDGE_SUBSCRIPTIONS` | `subscriptions` | The file that keeps the devices' push subscriptions. In the settings file, an absolute path | `~/Library/Application Support/nudge/subscriptions.json` |

The settings file suits `brew services`, which starts `nudge serve` without the shell's variables ([Run at login](#run-at-login)). It holds a JSON object:

```json
{
  "login": "user@example.com",
  "host": "machine.tailnet.ts.net"
}
```

Tailscale knows the login and the name. This writes them to the file, in a folder that only the user can open, as `nudge serve` needs:

```sh
mkdir -p -m 700 ~/Library/Application\ Support/nudge
tailscale status --json |
  jq '{login: .User[(.Self.UserID | tostring)].LoginName, host: (.Self.DNSName | rtrimstr("."))}' \
  > ~/Library/Application\ Support/nudge/settings.json
```

A missing file is fine. Invalid JSON, an unknown key, a key twice, a value that isn't a string, or a `subscriptions` path that isn't absolute stops `nudge serve` with a message that names the file.

The variables suit a run from the checkout. They live in `mise.local.toml`, at the repo's root: git ignores it, and mise puts its `[env]` into every task's environment, so `mise run serve` gets them.

```toml
[env]
NUDGE_LOGIN = "user@example.com"
NUDGE_HOST = "machine.tailnet.ts.net"
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

To try nudge, run `nudge serve` in a shell:

```sh
nudge serve
```

`nudge serve` runs until Ctrl-C. It logs each refused request to stderr, with the login and the host it came with. In another shell, ask a question:

```sh
nudge ask --option Yes --option No "Merge the pull request?"
```

Answer it on the page: tap an option, send a reply, or type a reply and then tap an option to send both. `nudge ask` prints the answer and exits 0.

In the checkout, `mise run serve` and `go run . ask` do the same from the source.

## Run at login

Once nudge is installed ([Install](#install)) and the settings file holds its settings ([Settings](#settings)), start the service:

```sh
brew services start codedogapp/tap/nudge
```

It writes the launchd agent `~/Library/LaunchAgents/sh.brew.nudge.plist` and loads it. launchd starts `nudge serve` at once and at each login, and starts it again whenever it stops. It starts it with none of the shell's variables, so `nudge serve` takes its settings from the settings file. `tailscale serve` must stay on ([Tailscale](#tailscale)).

The service's log is `$(brew --prefix)/var/log/nudge.log`, which gets the refused requests and the errors, and which nothing rotates. When `nudge serve` can't start, launchd tries again every 10 seconds, and the log says why.

The service keeps the binary and the settings it started with: restart it after `brew upgrade` or a settings change. A `nudge serve` in a shell can't run next to the service, which holds the port and the socket. To restart the service, to see it, and to stop it and keep it from starting at login:

```sh
brew services restart codedogapp/tap/nudge
brew services info codedogapp/tap/nudge
brew services stop codedogapp/tap/nudge
```

## The VAPID key

`nudge serve` signs its Web Pushes with a VAPID key. On its first start, it makes one and adds it to the login Keychain as service `nudge`, account `vapid-private-key`; later starts read it from there ([the plan](plan.md), decision 11). To print it:

```sh
security find-generic-password -s nudge -a vapid-private-key -w
```

`nudge serve` never replaces the key. For a new one, delete it and start `nudge serve` again; every device must then turn its notifications on again:

```sh
security delete-generic-password -s nudge -a vapid-private-key
```

With `NUDGE_VAPID_KEY_FILE` set, the key lives in that file instead, as the private key in base64url without padding, and `nudge serve` doesn't touch the Keychain. A file that isn't a key stops `nudge serve`, which never replaces it.

When `security` fails, for example on a locked Keychain that can't ask for its password, or the stored key isn't valid, `nudge serve` exits 1 and says why.

## The subscriptions

Each device that turns its notifications on adds its push subscription to `~/Library/Application Support/nudge/subscriptions.json`, or to the file that the `subscriptions` setting names: a JSON list that only the user can read. To see it:

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

An agent asks through `nudge mcp`, an MCP server with two tools, `ask`, which waits for your answer ([the plan](plan.md), decision 17), and `report`, which sets a note on the session that the page shows ([the plan](plan.md), decision 31). The agent starts the server itself, from the binary that Homebrew installed ([Install](#install)). Add it to Copilot CLI, with a timeout of a day:

```sh
copilot mcp add --timeout 86400000 nudge -- "$(brew --prefix codedogapp/tap/nudge)/bin/nudge" mcp
```

Keep the name `nudge`: the agent sees the tool as `nudge-ask`, and the skill calls it that and approves it as `nudge(ask)`; the same goes for `nudge-report` and `nudge(report)` ([The skill](#the-skill)).

Copilot CLI ends a tool call that runs longer than the timeout, and the question goes away with it. Without `--timeout`, Copilot CLI 1.0.102 ended a question after three minutes (risk 9). `copilot mcp add` keeps the server in `~/.copilot/mcp-config.json`, and `copilot mcp get nudge` shows it.

Copilot CLI starts `nudge mcp` in the session's folder, so a question's project is that folder's name. `nudge mcp` reaches `nudge serve` through the same socket as `nudge ask`: when `NUDGE_SOCKET` isn't the default, add `--env NUDGE_SOCKET=<path>` before the `--`.

Copilot CLI can ask at the Mac for permission before the agent uses the tool, which defeats the point when you're away. The skill lets the agent ask without that prompt once it's loaded ([The skill](#the-skill)). Without the skill, start Copilot CLI with `--allow-tool 'nudge(ask)'`.

## The skill

The `nudge` skill, `skills/nudge/SKILL.md` in the repo, tells a Copilot CLI session to ask every question through nudge, never with `ask_user` or in the chat, to report its progress with `nudge-report`, and to run `nudge ask` when the MCP server isn't there ([the plan](plan.md), decision 23). A session loads it only when you ask for it: before you step away, start a prompt with `/nudge`, as in `/nudge Fix the failing test`. A prompt that asks the agent to ask you on your phone may load it too, but that's up to the model. It holds until you tell the agent you're back.

Homebrew installs the skill with nudge. Register its folder with Copilot CLI:

```sh
copilot skill add "$(brew --prefix codedogapp/tap/nudge)/share/nudge/skills"
```

`copilot skill list` then shows `nudge`, and `/skills info nudge` in a session shows its details. A session that is already running finds it after `/skills reload`. Copilot CLI reads the skill from the folder each time, through a path that an upgrade keeps, so a new version's skill reaches the next session ([the plan](plan.md), fact 13).

Once a session loads the skill, the agent may use the `ask` tool and run `nudge` in the shell without asking you for permission first: the skill's `allowed-tools` approves `nudge(ask)`, `nudge(report)` and `shell(nudge:*)` for the rest of the session.
