# Remember Me MCP - Ansible Deployment

This directory contains Ansible playbooks and roles for deploying the Remember Me MCP service to production servers.

## Architecture

The deployment sets up:
- PostgreSQL 15 with pgvector extension
- Remember Me MCP HTTP API server (Go binary)
- Caddy reverse proxy with automatic HTTPS
- UFW firewall (only ports 22, 80, 443 open)
- Fail2ban for intrusion prevention
- Automated backups and log rotation

## Prerequisites

1. **Control Machine Requirements:**
   - Ansible 2.10+ installed
   - SSH access to target servers
   - ansible-vault for managing secrets

2. **Target Server Requirements:**
   - Ubuntu 20.04 LTS or newer
   - Minimum 2GB RAM, 2 CPU cores
   - Domain name pointing to server IP (for HTTPS)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/ksred/remember-me-mcp.git
cd remember-me-mcp/ansible
```

### 2. Install Ansible Dependencies

```bash
pip install ansible ansible-vault
```

### 3. Configure Inventory

Edit the inventory file for your environment:

```bash
# For production
cp inventory/production inventory/production.local
vim inventory/production.local

# Update the following:
# - Replace YOUR_PRODUCTION_IP with your server's IP
# - Update ansible_user if not using ubuntu
# - Update SSH key path if needed
```

### 4. Create Vault File

```bash
# Copy the example vault file
cp group_vars/all/vault.yml.example group_vars/all/vault.yml

# Edit with your secrets
ansible-vault edit group_vars/all/vault.yml

# Set the following:
# - vault_postgres_password: Strong password for PostgreSQL
# - vault_openai_api_key: Your OpenAI API key
# - vault_admin_email: Admin email for notifications
# - vault_jwt_secret: Random secret for JWT tokens
# - vault_caddy_email: Email for Let's Encrypt certificates
```

### 5. Deploy

```bash
# Deploy to production
ansible-playbook -i inventory/production playbooks/deploy.yml --ask-vault-pass

# Deploy to staging
ansible-playbook -i inventory/staging playbooks/deploy.yml --ask-vault-pass
```

## Playbooks

### deploy.yml - Full Deployment
Deploys the complete stack including all dependencies:
```bash
ansible-playbook -i inventory/production playbooks/deploy.yml --ask-vault-pass
```

### update.yml - Update Application
Updates only the Remember Me application (pulls latest code, rebuilds):
```bash
ansible-playbook -i inventory/production playbooks/update.yml --ask-vault-pass
```

### rollback.yml - Rollback Application
Rollbacks to a previous version:
```bash
ansible-playbook -i inventory/production playbooks/rollback.yml --ask-vault-pass
# You'll be prompted for the backup timestamp to rollback to
```

## Configuration

### Environment-Specific Variables

- `group_vars/all/main.yml` - Common variables for all environments
- `group_vars/production/main.yml` - Production-specific settings
- `group_vars/staging/main.yml` - Staging-specific settings

### Secrets Management

All sensitive data is stored in encrypted vault files:
```bash
# View vault contents
ansible-vault view group_vars/all/vault.yml

# Edit vault contents
ansible-vault edit group_vars/all/vault.yml

# Change vault password
ansible-vault rekey group_vars/all/vault.yml
```

## SSL/TLS Certificates

Caddy automatically obtains and renews Let's Encrypt certificates. For production:
- Ensure your domain points to the server IP
- Certificates are stored in `/var/lib/caddy/certificates`
- Automatic renewal happens 30 days before expiry

## Monitoring and Logs

### Application Logs
- Remember Me API: `/var/log/remember-me/server.log`
- Caddy access logs: `/var/log/caddy/access.log`
- PostgreSQL logs: `/var/log/postgresql/`

### View Logs
```bash
# On the server
sudo journalctl -u remember-me -f  # Application logs
sudo journalctl -u caddy -f         # Caddy logs
sudo tail -f /var/log/remember-me/server.log
```

### Health Checks
- Application health: `https://api.remembermcp.com/health`
- Metrics (protected): `https://api.remembermcp.com/metrics`

## Backup and Recovery

### Database Backups
- Automatic daily backups at 3 AM
- Stored in `/var/backups/postgresql/`
- 7 days retention

### Manual Backup
```bash
# On the server
sudo -u postgres /usr/local/bin/backup-remember-me-db
```

### Restore from Backup
```bash
# On the server
sudo -u postgres psql remember_me < /var/backups/postgresql/remember_me_20240115_030000.sql.gz
```

## Security

### Firewall Rules
- Port 22: SSH (rate limited)
- Port 80: HTTP (redirects to HTTPS)
- Port 443: HTTPS
- All other ports: Blocked

### Fail2ban Protection
- SSH: 3 failed attempts = 1 hour ban
- API: 10 failed auth attempts in 5 minutes = 30 minute ban

## Troubleshooting

### Service Issues
```bash
# Check service status
sudo systemctl status remember-me
sudo systemctl status caddy
sudo systemctl status postgresql

# Restart services
sudo systemctl restart remember-me
sudo systemctl restart caddy
```

### Database Connection
```bash
# Test database connection
sudo -u postgres psql -d remember_me -c "SELECT 1;"

# Check pgvector extension
sudo -u postgres psql -d remember_me -c "SELECT * FROM pg_extension WHERE extname = 'vector';"
```

### Common Issues

1. **Deployment fails at PostgreSQL step**
   - Ensure server has at least 2GB RAM
   - Check if PostgreSQL 15 repository is accessible

2. **Caddy fails to obtain certificates**
   - Verify domain DNS points to server IP
   - Check firewall allows ports 80/443
   - For staging, self-signed certs are expected

3. **API returns 502 Bad Gateway**
   - Check if Remember Me service is running
   - Verify service is listening on correct port
   - Check application logs for errors

## Advanced Usage

### Custom Deployment
```bash
# Deploy specific roles only
ansible-playbook -i inventory/production playbooks/deploy.yml --tags "postgresql,caddy"

# Skip certain roles
ansible-playbook -i inventory/production playbooks/deploy.yml --skip-tags "firewall"

# Dry run
ansible-playbook -i inventory/production playbooks/deploy.yml --check
```

### Multiple Environments
```bash
# Deploy to specific hosts
ansible-playbook -i inventory/production playbooks/deploy.yml --limit "api.remembermcp.com"

# Deploy with custom variables
ansible-playbook -i inventory/production playbooks/deploy.yml -e "remember_me_port=8083"
```

## Maintenance

### Update System Packages
```bash
ansible all -i inventory/production -m apt -a "upgrade=yes update_cache=yes" --become
```

### Rotate Logs Manually
```bash
ansible all -i inventory/production -m command -a "logrotate -f /etc/logrotate.d/remember-me" --become
```

### Check Disk Usage
```bash
ansible all -i inventory/production -m command -a "df -h" --become
```

## Support

For issues or questions:
1. Check application logs first
2. Review this documentation
3. Open an issue on [GitHub](https://github.com/ksred/remember-me-mcp/issues)