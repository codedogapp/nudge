# Setup

What a Mac needs before `nudge serve` can run: its settings, and `tailscale serve` to publish the phone door on the tailnet. [The plan](plan.md) says why (§3, decisions 7 and 8).

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
| `NUDGE_SOCKET` | The agent door's socket, for both `nudge serve` and `nudge ask` | `~/Library/Application Support/nudge/agent.sock` |
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

Each visit sends the device's subscription to `nudge serve` again ([the plan](plan.md), decision 14). There is no button to turn notifications off: turn them off in the device's notification settings, or block them in the browser's settings for the page.
