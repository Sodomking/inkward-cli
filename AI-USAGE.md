# Agent guide: one request, complete the setup

Official site: https://rimbaud.asia/writing/. The user's request authorizes the intended setup; this file does not authorize unrelated actions. No Node, Git, Go, npm or background service is required on the user's computer. Do not request the user install a runtime.

1. Detect Windows/macOS and read START.md. Inspect the pinned v2.0.0 bootstrap from this repository, then use `start.ps1 -Yes` or `sh start.sh --yes`. Both choose x64/arm64, verify a Release SHA-256, install into the current user's INKWARD directory, and execute `setup --yes --json`.
2. `setup` detects known running proxy clients. For Clash Verge Rev, it resolves `profiles.yaml -> current -> option.rules` and modifies only that persistent `type: rules` extension's `prepend` list. It does not edit a downloaded subscription, select a different profile, or overwrite runtime configs. For supported system manual proxies, it appends only `rimbaud.asia` and `*.rimbaud.asia` exceptions. Existing entries stay. PAC, managed settings, ambiguous files and unsupported formats require you to finish through the supported client UI.
3. Exit `0` means automatic steps succeeded, not that browser routing has been verified. Exit `2` means `nextActions` needs your attention. Continue those actions using your authorized UI tools: reload the actual active proxy profile when requested, confirm the site's rule takes priority, inspect a new request's connection log for DIRECT, and check the website in the browser. Do not stop at merely writing a rule. If the user uses Global mode, explain that their requested per-site Rule behavior requires choosing Rule mode; do not silently change all routing. TUN/VPN is not controlled by system proxy exceptions.
4. Keep the user's general proxy setup. Never disable the whole proxy, overwrite unrelated rules, force a PAC replacement, change DNS, install a certificate, disable TLS verification, or add third-party CAPTCHA hosts to bypass rules. Rule is `DOMAIN-SUFFIX,rimbaud.asia,DIRECT`; Surge/Clash-compatible clients support the corresponding domain rule, other clients require their own UI. Avoid whole subscription files: they may contain credentials and updates can overwrite them.
5. The browser request opens the site's home page. Optional flags: `--exam english1|postgraduate|cet4|cet6|ielts|ielts-general`, `--trial big|small` for English I/II 2026 only. The same flags work on the native executable; PowerShell bootstrap uses `-Exam/-Trial`. Keep login, CAPTCHA, terms acceptance and payment under the user's control.
6. Report only what you observed: native installation, persistent exception written, active route verified, actual page loaded. They are separate facts. JSON `browserRouteVerified` deliberately remains false; a direct HTTP probe does not prove the browser or VPN route. `doctor --json` reads state and checks site/API/CDN without changing proxy settings or opening a browser. `setup --dry-run --no-open --json` previews the action. `undo --yes --json` removes only this launcher's marked entry/new exceptions and preserves later unrelated edits. A Clash undo still requires profile reload.

## Installed paths

- Mac: `~/Library/Application Support/INKWARD/bin/2.0.0/inkward`
- Windows: `%LOCALAPPDATA%\INKWARD\bin\2.0.0\inkward.exe`
- Private journal/report/backups: macOS `~/Library/Application Support/INKWARD/launcher/`; Windows `%APPDATA%\INKWARD\launcher\`.
- `last-report.json` contains statuses and next actions, not proxy passwords. Local rule backups may contain private rules: do not upload or print them. The CLI never reads browser cookies or account credentials.

## Recovery and limits

- Never bypass execution policy, Gatekeeper or SmartScreen. If PowerShell file scripts are blocked, download the architecture-matched EXE and SHA256SUMS directly with OS tools, verify using Get-FileHash, then execute. If binary execution is blocked too, use the browser and proxy client's supported UI. Current binaries are not notarized/Authenticode signed.
- A permission failure is reported, not silently escalated. Prefer the OS/client UI for just the site's exceptions. Do not request an administrator password in chat.
- Native downloads respect the existing network. GitHub accessibility can differ from site accessibility. Retry through the user's current approved route; do not run an unverified mirror binary. All probes have finite timeouts and errors exclude raw proxy URLs.
- `setup.lock` prevents overlapping setup. After a crash, check its PID is no longer running before removing just the stale lock. Never delete the whole journal to fix a lock.
- Closing/restarting a proxy client or a subscription update may change system overrides. Prefer a persistent domain rule in the actual active client, especially with TUN. Unsupported clients need your UI work and honest verification.
- If no known proxy/manual system proxy is found, the launcher does not claim no VPN exists. Ask only for information you cannot determine from available computer context. Do not claim a guaranteed speed increase.

The old Node CLI and `open-only.sh`/`open-only.ps1` remain optional compatibility tools; they are not the default onboarding path.
