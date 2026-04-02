# shhgit Server Deployment Guide

This guide covers deploying shhgit on a virtual server or VPS.

## Requirements

- Linux server (Ubuntu/Debian recommended)
- Root or sudo access
- At least 2GB RAM
- At least 10GB free disk space
- Internet connection

## 1. Install Docker and Docker Compose

### Ubuntu/Debian:

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Add user to docker group (optional, to run without sudo)
sudo usermod -aG docker $USER
# Note: Log out and back in for this to take effect
```

### Verify installation:

```bash
docker --version
docker compose version
```

## 2. Transfer Project to Server

### Option 1: Clone with Git (recommended)

```bash
git clone https://github.com/mstfknn/shhgit.git
cd shhgit
```

### Option 2: Manual file transfer

```bash
# From local machine via SCP
scp -r /path/to/shhgit user@server-ip:/home/user/
```

## 3. Configure

### Create `.env` file with your tokens:

```bash
cat > .env << 'EOF'
GITHUB_TOKEN_1=ghp_your_first_token_here
GITHUB_TOKEN_2=ghp_your_second_token_here
EOF
```

### Optional: Configure webhook in `config.yaml`

```yaml
webhook: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
```

## 4. Port Configuration

Default web interface port is **8080**. To change it, edit `docker-compose.yml`:

```yaml
services:
  shhgit-www:
    ports:
      - "80:80"    # Use port 80 (requires root)
      # or
      - "3000:80"  # Use port 3000
```

## 5. Firewall Rules

### UFW (Ubuntu Firewall):

```bash
# Open port 8080 (or your chosen port)
sudo ufw allow 8080/tcp
sudo ufw status
```

### iptables:

```bash
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables-save
```

## 6. Start the Project

### Initial setup:

```bash
cd shhgit

# Build Docker images
docker compose build

# Start containers
docker compose up -d

# Check logs
docker compose logs -f
```

### Stop containers:

```bash
docker compose down
```

### Restart containers:

```bash
docker compose restart
```

### View logs:

```bash
# All logs
docker compose logs -f

# App logs only
docker compose logs -f shhgit-app

# Web logs only
docker compose logs -f shhgit-www
```

## 7. Access and Test

### Local network access:

Open in your browser:

```
http://server-ip-address:8080
```

### External access (optional):

1. **Router port forwarding**: Forward port 8080 to your server's local IP
2. **Dynamic DNS**: Use a service like DynDNS for a stable hostname
3. **VPN**: Use a VPN for secure remote access

## 8. Run as a System Service (Optional)

Create a systemd service for automatic startup:

```bash
sudo nano /etc/systemd/system/shhgit.service
```

Contents:

```ini
[Unit]
Description=shhgit Docker Compose
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/shhgit
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

Enable the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shhgit.service
sudo systemctl start shhgit.service
```

## 9. Disk Space Management

The project continuously clones repositories — monitor disk usage:

```bash
# Check disk usage
df -h

# Check temp directory inside container
docker exec shhgit.app du -sh /tmp/shhgit

# Clean up if needed
docker exec shhgit.app sh -c "rm -rf /tmp/shhgit/*"
```

The project automatically deletes processed repositories, but occasional manual cleanup may be needed.

## 10. Troubleshooting

### Containers not running:

```bash
docker compose ps
docker compose logs
docker compose restart
```

### Port already in use:

```bash
sudo netstat -tulpn | grep 8080
# or
sudo ss -tulpn | grep 8080
```

### Disk space full:

```bash
# Clean up Docker images
docker system prune -a

# Clean up unused volumes
docker volume prune
```

### GitHub API rate limit:

- Add more GitHub tokens to `.env`
- Verify tokens have appropriate permissions
- Check rate limit logs: `docker compose logs shhgit-app | grep -i rate`

## 11. Security Recommendations

1. **Firewall**: Only open necessary ports
2. **SSH**: Use key-based authentication
3. **GitHub Tokens**: Rotate tokens regularly
4. **Updates**: Keep system and Docker updated
5. **Backup**: Back up your `.env` and `config.yaml` files

## 12. Performance Tuning

### Adjust thread count in `docker-compose.yml`:

```yaml
shhgit-app:
  entrypoint: ["/app/shhgit", "--live=http://shhgit-www/push", "--threads=8"]
```

Set the thread count based on your server's CPU cores.

## Quick Start Summary

```bash
# 1. Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
sudo apt install docker-compose-plugin -y

# 2. Clone project
git clone https://github.com/mstfknn/shhgit.git
cd shhgit

# 3. Configure tokens
cat > .env << 'EOF'
GITHUB_TOKEN_1=ghp_your_token_1
GITHUB_TOKEN_2=ghp_your_token_2
EOF

# 4. Open firewall port (if needed)
sudo ufw allow 8080/tcp

# 5. Build and start
docker compose build
docker compose up -d

# 6. Verify
docker compose ps
docker compose logs -f
```
