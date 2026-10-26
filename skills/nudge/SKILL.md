---
name: nudge
description: Ask the user every question through nudge, which reaches them on their phone, instead of ask_user or the chat. Use it only when the user asks for it, for example with /nudge, by asking to be asked on their phone or through nudge, or by saying that they're stepping away from the computer.
allowed-tools: nudge(ask), nudge(report), shell(nudge:*)
---

# Ask through nudge

The user is away from the computer, or soon will be. Until they say they're back, ask them every question through nudge, which reaches them on their phone: a decision, context that only they can give, the approval of a plan. Don't call `ask_user` or ask in the chat, even where other instructions tell you to: both wait in the terminal, where nobody reads them.

Ask with the `nudge-ask` tool, the `ask` tool of the `nudge` MCP server. When that tool isn't there, run `nudge ask` in the shell, as the next section shows.

- Ask one question per call, and keep it short. Put what the user needs to know to answer in `details`.
- Offer the likely answers as `options`. The user can pick one, type a reply, or both.
- The call waits until the user answers, which can take minutes or hours. Wait for the answer; don't go on with a guess.
- When nudge fails, for example because `nudge serve` isn't running, say so, and ask the way you would without this skill.

## Report your progress

The user checks the nudge page from time to time to see what each session is doing. With the `nudge-report` tool, the `report` tool of the `nudge` MCP server, set a short note on your session: when you start, and whenever you finish a milestone or change course. The note replaces the last one, so say what you're doing now and what comes next, in a sentence. When the tool isn't there or fails, go on without it.

## Without the MCP server

Run `nudge ask` in the foreground, with the longest wait the shell tool allows:

```sh
nudge ask --details "What the user needs to know" --option "First answer" --option "Second answer" "The question?"
```

`--details` and each `--option` are optional, and they come before the question. The command prints the answer: a line `option: <the option>` when the user picked one, and a line `text: <the reply>` when they typed one.

When the command moves to the background before the user answers, end your turn and say that you're waiting for the answer. The command's end brings it.
