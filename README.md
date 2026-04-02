<p align="center">
<img src="./images/logo.png" height="30%" width="30%" />

# shhgit v0.5

## **Find secrets in real time across GitHub, GitLab and BitBucket — before they lead to a security breach.**

> This is a maintained fork of the original [eth0izzle/shhgit](https://github.com/eth0izzle/shhgit) which is no longer maintained. This fork includes 300+ detection signatures, security hardening, a modern dashboard, and production-ready deployment.

</p>

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

- **Real-time scanning** — monitors GitHub events, Gists, and issue comments as they happen
- **300 detection signatures** — API keys, tokens, private keys, credentials, database files, cloud secrets, and more
- **Modern dashboard** — responsive UI with live updates, signature filtering, match counts, and browser notifications
- **Local mode** — scan local directories, integrate into CI pipelines
- **Custom search** — use regex to find specific patterns: `--search-query AWS_ACCESS_KEY_ID=AKIA`
- **Webhook support** — POST match events to Slack, Mattermost, or any HTTP endpoint
- **CSV export** — write findings to CSV for further analysis
- **Security hardened** — XSS protection, CSP headers, non-root Docker, SRI integrity checks

## Signature Coverage

300 signatures organized across these categories:

| Category | Count | Examples |
|----------|-------|---------|
| API Keys & Tokens | 80+ | OpenAI, Anthropic, Groq, Mistral, Stripe, SendGrid, Twilio |
| Cloud Infrastructure | 25+ | AWS (Access Key, Secret, Session, RDS, S3), Azure, GCP, Supabase, Cloudflare |
| Modern DevOps | 30+ | Vercel, PlanetScale, Turso, Fly.io, Railway, Neon, Terraform, Tailscale |
| Package Registries | 5+ | npm, PyPI, RubyGems |
| Developer Tools | 15+ | GitHub (PAT, fine-grained, ghu_, ghs_, gho_, ghr_), GitLab, Postman, Figma, Databricks |
| Monitoring | 5+ | Grafana Cloud, Grafana Service Account, Doppler |
| SSH/Crypto Keys | 45+ | PEM, PKCS12, PFX, RSA, DSA, ED25519, ECDSA, PGP |
| Database Credentials | 30+ | PostgreSQL, MySQL, MongoDB, Redis, Elasticsearch, Neon connection strings |
| Config Files | 40+ | .env, .npmrc, .bashrc, database.yml, wp-config.php, Django settings |
| Authentication | 25+ | OAuth, JWT, Bearer tokens, Session IDs, CSRF tokens |

### Recently Added (2024-2025 services)

Vercel, Supabase PAT, PlanetScale, Turso, Fly.io, Railway, Cloudflare API Token, Neon DB, OpenAI Service Account (`sk-svcacct-`), xAI/Grok (`xai-`), npm, PyPI, RubyGems, Postman, Pulumi, Databricks, Figma, Sourcegraph, Grafana Cloud, Doppler, Terraform Cloud, Tailscale, Deno Deploy, Resend, Infracost, Prefect, Buildkite

---

## What's Changed in This Fork

### Security Hardening
- XSS protection — all match data sanitized on backend (`html.EscapeString`) and frontend (`escapeHtml`)
- `/push` endpoint restricted to internal Docker network only
- Content Security Policy, X-Frame-Options, X-Content-Type-Options, Referrer-Policy headers
- SRI integrity hashes on all third-party scripts and stylesheets
- CORS wildcard removed from `/matches.jsonl`
- Docker containers run as non-root with `no-new-privileges` and `cap_drop: ALL`
- GitHub tokens moved to `.env` file (never committed to repo)
- Push server hardened: 64KB request limit, file locking, no error detail leakage

### Bug Fixes
- Fixed `http.Post()` response body leak in publish function
- Fixed `context.WithTimeout` leak in goroutine (deferred cancel never executed)
- Fixed `ProcessComments` missing `defer os.RemoveAll` (temp files left on disk after panic)
- Fixed frontend source enum mapping (Local, GitHub Comment were missing)
- Removed deprecated `rand.Seed` (Go 1.20+)
- Regex search query compiled once at startup instead of per-file (`regexp.Compile` vs `MustCompile`)
- Removed duplicate Twilio signature

### Signature Improvements
- Added 33 new high-confidence prefix-based signatures (see list above)
- Added 4 missing GitHub token variants (`ghu_`, `ghs_`, `gho_`, `ghr_`)
- Reduced false positives: Session ID min length 10 to 20, OAuth Client ID min 10 to 30, Base64 entropy min 40 to 60
- Blacklist filtering applied before publishing events
- High-entropy strings classified as "API Key" when context keywords found

### Frontend & Dashboard
- Bulma CSS removed (unused 200KB dependency)
- Modern, accessible UI with WCAG 2.2 compliance
- Skip-to-content link for keyboard navigation
- ARIA labels, roles, and live regions throughout
- SRI integrity verified third-party assets
- Non-render-blocking font loading
- Row fade-in animations with `prefers-reduced-motion` support
- Empty state and loading indicators
- Gradient-coded signature badges (top 5 ranked by count)
- Print stylesheet
- `aria-expanded` on mobile menu toggle
- 0 console errors

### Infrastructure
- Polling-based frontend (replaced PushStream/EventSource)
- Duplicate match filtering on both backend and frontend
- Nginx access/error logging enabled
- Comment temp files written with `0600` permissions
- Alpine 3.19 pinned (not `latest`)
- `docker-compose.yml` with `security_opt` and `cap_drop`

---

## Architecture

```
[GitHub/GitLab/BitBucket Events]
         |
    [shhgit-app]          Go backend — clones repos, scans files, matches 300 signatures
         |
    HTTP POST /push       (restricted to Docker internal network)
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
