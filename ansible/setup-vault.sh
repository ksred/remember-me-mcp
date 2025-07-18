#!/bin/bash
# Setup Ansible Vault for Remember Me MCP

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Remember Me MCP - Vault Setup${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# Check if vault file already exists
if [ -f "group_vars/all/vault.yml" ]; then
    echo -e "${YELLOW}Warning: vault.yml already exists${NC}"
    read -p "Do you want to overwrite it? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Setup cancelled${NC}"
        exit 0
    fi
fi

# Copy example file
cp group_vars/all/vault.yml.example group_vars/all/vault.yml

echo -e "${GREEN}Vault file created from example${NC}"
echo ""
echo -e "${YELLOW}Please update the following values in vault.yml:${NC}"
echo "  - vault_postgres_password: Database password"
echo "  - vault_openai_api_key: Your OpenAI API key"
echo "  - vault_admin_email: Admin email for notifications"
echo "  - vault_jwt_secret: Random JWT secret"
echo "  - vault_caddy_email: Email for Let's Encrypt"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Edit the vault file:"
echo "   ansible-vault edit group_vars/all/vault.yml"
echo ""
echo "2. Set a strong vault password when prompted"
echo ""
echo "3. (Optional) Create a password file for automation:"
echo "   echo 'your-vault-password' > ~/.vault_pass"
echo "   chmod 600 ~/.vault_pass"
echo ""
echo "4. Deploy with:"
echo "   ./deploy.sh -e staging"
echo "   or"
echo "   ./deploy.sh -e production -v ~/.vault_pass"
echo ""
echo -e "${GREEN}Setup complete!${NC}"