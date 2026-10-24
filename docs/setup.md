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
| `NUDGE_SOCKET` | The agent door's socket, for both `nudge serve` and `nudge ask` | `~/Library/Application Support/nudge/agent.sock` |

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
