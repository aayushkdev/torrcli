# torrcli

A cross-platform terminal BitTorrent client written in Go.

The project has two executables:

- `torrcli` — the terminal interface and command-line client.
- `torrd` — the background daemon that manages downloads and seeding.

`torrcli` communicates with `torrd` locally, so transfers can continue after
the TUI closes when the daemon is managed by the operating system.

Current commands:

```text
torrcli status
torrcli add SOURCE --save-path PATH
torrcli list
torrcli pause ID
torrcli resume ID
torrcli remove ID
torrcli remove ID --delete-data
torrcli priority ID FILE_INDEX skip|normal|high
```