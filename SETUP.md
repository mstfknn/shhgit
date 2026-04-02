# shhgit Quick Setup Guide

## Quick Start

### 1. Connect to Server

```bash
ssh user@server-ip
```

### 2. Install Requirements

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo apt install docker-compose-plugin -y
sudo usermod -aG docker $USER
```

**Important:** After adding to the docker group, log out and back in or run `newgrp docker`.

### 3. Get the Project

**Option A: Clone with Git (recommended)**

```bash
git clone https://github.com/mstfknn/shhgit.git
cd shhgit
```

**Option B: File transfer**

```bash
# From local machine
scp -r /path/to/shhgit user@server:/home/user/
```

### 4. Automated Setup (Recommended)

```bash
cd shhgit
chmod +x deploy.sh
./deploy.sh
```

### 5. Manual Setup

**Create `.env` with your tokens:**

```bash
cat > .env << 'EOF'
GITHUB_TOKEN_1=ghp_your_token_1
GITHUB_TOKEN_2=ghp_your_token_2
EOF
```

**Start the project:**

```bash
docker compose build
docker compose up -d
```

### 6. Access

Open in your browser:

```
http://server-ip:8080
```

## Configuration

### Change Port

In `docker-compose.yml`:

```yaml
ports:
  - "80:80"    # Port 80 (requires root)
  # or
  - "3000:80"  # Port 3000
```

### Firewall Rules

```bash
# If using UFW
sudo ufw allow 8080/tcp
sudo ufw reload
```

## Common Commands

```bash
# Container status
docker compose ps

# View logs
docker compose logs -f

# App logs only
docker compose logs -f shhgit-app

# Restart containers
docker compose restart

# Stop containers
docker compose down
```

## Troubleshooting

### Containers not running

```bash
docker compose ps
docker compose logs
docker compose restart
```

### Port already in use

```bash
sudo netstat -tulpn | grep 8080
# Find and stop the process using the port
```

### Disk space

```bash
# Check disk usage
df -h

# Docker cleanup
docker system prune -a
```

## Important Notes

1. **GitHub Tokens**: At least 2 tokens recommended (for rate limiting)
2. **Disk Space**: The project continuously clones repositories, monitor disk usage
3. **Security**: Review your firewall rules
4. **Backup**: Back up your `.env` and `config.yaml` files

## Detailed Guide

For more information, see `DEPLOYMENT.md`.
