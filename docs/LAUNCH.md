# Launch kit

## One-line description

Process Guard explains and safely stops dev servers left running by coding
agents, with exact Herdr workspace, tab, and process-tree attribution.

## Suggested launch title

> My coding agents kept leaving dev servers behind, so I built a process
> inspector that remembers which agent started each one

## Suggested short post

Coding agents regularly start Vite, Python, Node, browser, and test processes.
When an agent becomes idle, it is difficult to tell whether a listening server
is intentional or forgotten—and `pkill` is a dangerous answer.

Process Guard is a macOS-first Herdr plugin that shows the exact agent, named
tab, parent lineage, activity reason, and complete stop blast radius. Discovery
is read-only; it never auto-kills; graceful and force stop are separate typed
confirmations.

Install:

```sh
herdr plugin install Efeguclu1/herdr-process-guard --yes
```

## Repository metadata

- Description: `Explain and safely stop dev servers left running by coding agents.`
- Website: `https://herdr.dev/`
- Social preview: `docs/assets/social-preview.png`
- Topics: `herdr`, `herdr-plugin`, `coding-agents`, `codex`, `claude-code`,
  `cursor`, `developer-tools`, `process-manager`, `dev-server`, `macos`,
  `orphan-process`

Upload the social preview through GitHub repository settings after the remote
repository exists. Enable Discussions and private vulnerability reporting.
