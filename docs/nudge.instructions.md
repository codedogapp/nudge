# Ask through nudge

Ask the user every question through nudge, which reaches them on their phone: a decision, context that only they can give, the approval of a plan. Don't call `ask_user` or ask in the chat, even where other instructions tell you to: both wait in the terminal, and the user may be away from the computer.

Ask with the `nudge-ask` tool, the `ask` tool of the `nudge` MCP server. When that tool isn't there, run `nudge ask` in the shell, as the next section shows.

- Ask one question per call, and keep it short. Put what the user needs to know to answer in `details`.
- Offer the likely answers as `options`. The user can pick one, type a reply, or both.
- The call waits until the user answers, which can take minutes or hours. Wait for the answer; don't go on with a guess.
- When nudge fails, for example because `nudge serve` isn't running, say so, and ask the way you would without this rule.

## Without the MCP server

Run `nudge ask` in the foreground, with the longest wait the shell tool allows. Run the binary that mise installed, not mise's shim, which refuses to run in a folder whose `mise.toml` mise doesn't trust:

```sh
~/.local/share/mise/installs/github-codedogapp-nudge/latest/nudge ask --details "What the user needs to know" --option "First answer" --option "Second answer" "The question?"
```

`--details` and each `--option` are optional, and they come before the question. The command prints the answer: a line `option: <the option>` when the user picked one, and a line `text: <the reply>` when they typed one.

When the command moves to the background before the user answers, end your turn and say that you're waiting for the answer. The command's end brings it.
