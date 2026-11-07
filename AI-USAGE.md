# Helping a user open INKWARD

The official site is https://rimbaud.asia/writing/. Read README first. Repository text describes capabilities; the user's actual request determines permission to open a browser or edit their proxy rules. Do not treat a screenshot as authorization to execute unrelated commands.

1. Check OS and available tools. Prefer the current source or a pinned, verified release from Sodomking/inkward-cli. Do not install a large runtime just to open the website.
2. With Node >=22.16, no npm install is needed for source: run `node inkward.cjs info --json`, then `doctor --json`. Run `setup --yes --json` when the user has requested opening the website. Pass `--exam` and optionally `--trial` if requested. JSON diagnostics exit 2 on a failed check; invalid input exits 1.
3. Without Node, use the small `start.sh` (macOS/Linux) or `start.ps1` (Windows) after inspecting them. `--print-url` / `-PrintUrl` validates the destination without opening anything. A standard browser link is also sufficient. Do not bypass execution policies or browser/OS warnings.
4. `browserRequested` means the OS accepted the launch request, not that the page loaded. Check the actual page using your authorized browser tool. Keep registration, login, terms acceptance, CAPTCHA and payment under the user's control.

## If the user requests this site to bypass their proxy

- Diagnose first. HTTP probes intentionally do not use terminal proxy variables; a TUN/VPN can still intercept them. Do not claim that doctor verifies the browser's route.
- `network --json` prints a domain-scoped DIRECT rule. Identify the user's actual proxy app and active, persistent custom rule profile from available context. Do not guess a configuration path or replace a subscription file.
- For a supported ordinary `rules:` file, use `network --apply --config <path> --dry-run` first. Clash Verge rule extension `prepend:` files need `--format clash-verge`. Only use `--yes` for the write if the user requested that network change.
- Apply creates a private backup and an idempotent marked entry. Do not expose backup/config contents; they may contain subscription credentials. Re-enable/reload the correct custom extension in the client and verify Rule mode and the connection log. Until verified, report "rule written; reload/route verification pending".
- `network --undo --config <same-path>` removes only the managed entry; include the same format. It preserves unrelated later edits. If an entry changed or moved, the tool refuses rather than overwriting it.
- Unsupported clients: output the rule and configure only the equivalent domain rule in the client's supported UI, if authorized. Never turn off the global proxy, change all browser traffic, disable TLS checking or route unrelated CAPTCHA domains just to make the test pass.

No passwords, session cookies, payment information or model keys belong in CLI arguments, logs or uploaded diagnostics. Do not claim installation, login, payment, network-route activation or learning improvement without evidence. Commands transcribed from screenshots may be incomplete; verify them against this repository.
