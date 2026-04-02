<p align="center">
<img src="./images/logo.png" height="30%" width="30%" />

# shhgit v0.5

## **Find secrets in real time across GitHub, GitLab and BitBucket — before they lead to a security breach.**

> This is a maintained fork of the original [eth0izzle/shhgit](https://github.com/eth0izzle/shhgit) which is no longer maintained. This fork includes security hardening, bug fixes, modern UI, and improved deployment.

</p>

## What's Changed in This Fork

Compared to the original project, this fork brings:

**Security**
- XSS protection — all match data is sanitized both on backend (`html.EscapeString`) and frontend (`escapeHtml`)
- `/push` endpoint restricted to internal Docker network only (no external write access)
- Content Security Policy, X-Frame-Options, X-Content-Type-Options headers added to Nginx
- CORS wildcard removed from `/matches.jsonl`
- Docker containers run as non-root with `no-new-privileges` and `cap_drop: ALL`
- GitHub tokens moved to `.env` file (out of config.yaml)
- Push server hardened: request size limits, file locking, no error detail leakage

**Bug Fixes**
- Fixed `http.Post()` response body leak in publish function
- Fixed `context.WithTimeout` leak in goroutine (deferred cancel never executed)
- Fixed `ProcessComments` missing `defer os.RemoveAll` (temp files left on disk after panic)
- Fixed frontend source enum mapping (was off by one — Local, GitHub Comment missing)
- Removed deprecated `rand.Seed` (Go 1.20+)
- Regex search query now compiled once at startup instead of per-file

**Improvements**
- Polling-based frontend (replaces PushStream/EventSource dependency)
- Duplicate match filtering on both backend and frontend
- Blacklist filtering applied before publishing events
- High-entropy strings classified as "API Key" when context keywords found
- Nginx access/error logging enabled for audit trail
- Comment temp files written with `0600` permissions instead of `0644`

---

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/mstfknn/shhgit.git
cd shhgit
```

Create a `.env` file with your GitHub tokens:

```bash
GITHUB_TOKEN_1=ghp_your_first_token_here
GITHUB_TOKEN_2=ghp_your_second_token_here
```

Tokens don't require any scopes — [create one here](https://github.com/settings/tokens).

### 2. Deploy

```bash
docker compose build
docker compose up -d
```

Open **http://localhost:8080** to access the dashboard.

### Automated Deployment

```bash
chmod +x deploy.sh
./deploy.sh
```

The script handles Docker installation, building, and container management.

### Local Scan (no tokens needed)

```bash
go build -o shhgit .
./shhgit --local /path/to/scan
```

---

## Features

- **Real-time scanning** — monitors GitHub events, Gists, and comments as they happen
- **800+ signatures** — API keys, tokens, private keys, credentials, database files, and more
- **Modern dashboard** — responsive UI with live updates, filtering, and notifications
- **Local mode** — scan local directories, integrate into CI pipelines
- **Custom search** — use regex to find specific patterns: `--search-query AWS_ACCESS_KEY_ID=AKIA`
- **Webhook support** — POST match events to Slack, Mattermost, or any HTTP endpoint
- **CSV export** — write findings to CSV for further analysis

## Architecture

```
[GitHub/GitLab/BitBucket Events]
         |
    [shhgit-app]          Go backend — clones repos, scans files, matches signatures
         |
    HTTP POST /push
         |
    [shhgit-www]          Nginx + Python push server
         |
    /tmp/matches.jsonl    Stored matches (last 1000)
         |
    [Browser]             Polls /matches.jsonl every 2s
```

## Options

```
--clone-repository-timeout   Max clone time in seconds (default 10)
--config-path                Directory to search for config.yaml
--csv-path                   Write findings to CSV file
--debug                      Print debug information
--entropy-threshold          Entropy threshold for secret detection (default 5.0, 0 to disable)
--live                       URL to POST match events (used by docker-compose)
--local                      Scan a local directory instead of public repos
--maximum-file-size          Max file size in KB (default 512)
--maximum-repository-size    Max repo size in KB (default 5120)
--minimum-stars              Min stars to scan (default 0)
--path-checks                Enable filename/path checks (default true)
--process-gists              Scan Gists in real time (default true)
--search-query               Custom regex search (ignores signatures)
--silent                     Suppress output except errors
--temp-directory             Temp directory for cloned repos
--threads                    Worker threads (default: CPU count)
```

## Config

The `config.yaml` supports environment variable expansion via `${VAR_NAME}`:

```yaml
github_access_tokens:
  - '${GITHUB_TOKEN_1}'
  - '${GITHUB_TOKEN_2}'
webhook: ''
webhook_payload: |
  {
    "text": "%s"
  }
blacklisted_strings: []
blacklisted_extensions: [".exe", ".jpg", ".png", ".gif", ".zip", ".tar.gz", ".lock"]
blacklisted_paths: ["node_modules{sep}", "vendor{sep}bundle"]
blacklisted_entropy_extensions: [".pem", "id_rsa", ".asc", ".ovpn"]
signatures:
  - part: 'extension'    # filename, extension, path, or contents
    match: '.pem'        # simple text match
    name: 'Potential cryptographic private key'
  - part: 'contents'
    regex: 'AKIA[0-9A-Z]{16}'  # regex pattern
    name: 'AWS Access Key ID'
```

## Troubleshooting

```bash
# Check status
docker compose ps

# View logs
docker compose logs -f shhgit-app
docker compose logs -f shhgit-www

# Restart
docker compose restart

# Rebuild
docker compose build && docker compose up -d
```

**Port conflict?** Change the port in `docker-compose.yml`:
```yaml
ports:
  - "3000:80"  # change 8080 to any available port
```

**Rate limited?** Add more tokens to `.env` and check logs:
```bash
docker compose logs shhgit-app | grep -i rate
```

## Credits

Originally created by [Paul Price (@darkp0rt)](https://github.com/eth0izzle/shhgit).
This fork is maintained by [Mustafa Kaan Demirhan (@mstfknn)](https://github.com/mstfknn).

## Disclaimer

This tool is for educational and authorized security testing purposes only. Use responsibly.

## License

MIT. See [LICENSE](https://github.com/mstfknn/shhgit/blob/master/LICENSE)
