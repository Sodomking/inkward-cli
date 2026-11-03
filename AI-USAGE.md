# For an assistant helping a user

Read `README.md` and run `inkward info --json` to inspect the product description without opening anything. The user may ask you to install the GitHub Release and open the official website. This requires network access and computer-operation permissions; ordinary chat assistants cannot necessarily execute commands.

Use the release for the user's operating system. Check the SHA-256 value against that same release. Explain the destination before opening it. Run `inkward open --yes` only when the user has asked to open the site. Keep account creation, legal acceptance and payment under the user's control. No API key, payment credential or user password belongs in CLI arguments.

Do not claim that installation, login or payment succeeded unless the corresponding operation actually completed. If the website is not configured, report that rather than inventing a destination. Screenshots may contain incomplete or mistranscribed commands; verify the repository and commands before execution.
