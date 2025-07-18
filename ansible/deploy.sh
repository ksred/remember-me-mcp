#!/bin/bash
# Remember Me MCP - Ansible Deployment Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT=""
VAULT_PASS_FILE=""
TAGS=""
CHECK_MODE=""

# Function to display usage
usage() {
    echo "Usage: $0 -e <environment> [-v <vault-password-file>] [-t <tags>] [-c] [-h]"
    echo ""
    echo "Options:"
    echo "  -e    Environment to deploy to (production/staging) [REQUIRED]"
    echo "  -v    Path to vault password file (optional)"
    echo "  -t    Ansible tags to run specific roles (optional)"
    echo "  -c    Run in check mode (dry run)"
    echo "  -h    Display this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -e production"
    echo "  $0 -e staging -v ~/.vault_pass"
    echo "  $0 -e production -t postgresql,caddy"
    echo "  $0 -e production -c"
    exit 1
}

# Parse command line arguments
while getopts "e:v:t:ch" opt; do
    case $opt in
        e)
            ENVIRONMENT=$OPTARG
            ;;
        v)
            VAULT_PASS_FILE=$OPTARG
            ;;
        t)
            TAGS=$OPTARG
            ;;
        c)
            CHECK_MODE="--check"
            ;;
        h)
            usage
            ;;
        \?)
            echo -e "${RED}Invalid option: -$OPTARG${NC}" >&2
            usage
            ;;
    esac
done

# Validate environment
if [ -z "$ENVIRONMENT" ]; then
    echo -e "${RED}Error: Environment is required${NC}"
    usage
fi

if [ "$ENVIRONMENT" != "production" ] && [ "$ENVIRONMENT" != "staging" ]; then
    echo -e "${RED}Error: Environment must be 'production' or 'staging'${NC}"
    usage
fi

# Check if inventory file exists
INVENTORY_FILE="inventory/$ENVIRONMENT"
if [ ! -f "$INVENTORY_FILE" ]; then
    echo -e "${RED}Error: Inventory file $INVENTORY_FILE not found${NC}"
    exit 1
fi

# Check if Ansible is installed
if ! command -v ansible-playbook &> /dev/null; then
    echo -e "${RED}Error: ansible-playbook not found. Please install Ansible.${NC}"
    exit 1
fi

# Install Ansible Galaxy requirements
echo -e "${YELLOW}Installing Ansible Galaxy dependencies...${NC}"
ansible-galaxy install -r requirements.yml

# Build ansible-playbook command
ANSIBLE_CMD="ansible-playbook -i $INVENTORY_FILE playbooks/deploy.yml"

# Add vault password file if provided
if [ -n "$VAULT_PASS_FILE" ]; then
    if [ ! -f "$VAULT_PASS_FILE" ]; then
        echo -e "${RED}Error: Vault password file $VAULT_PASS_FILE not found${NC}"
        exit 1
    fi
    ANSIBLE_CMD="$ANSIBLE_CMD --vault-password-file=$VAULT_PASS_FILE"
else
    ANSIBLE_CMD="$ANSIBLE_CMD --ask-vault-pass"
fi

# Add tags if provided
if [ -n "$TAGS" ]; then
    ANSIBLE_CMD="$ANSIBLE_CMD --tags=$TAGS"
fi

# Add check mode if requested
if [ -n "$CHECK_MODE" ]; then
    ANSIBLE_CMD="$ANSIBLE_CMD $CHECK_MODE"
    echo -e "${YELLOW}Running in CHECK MODE (dry run)${NC}"
fi

# Display deployment information
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Remember Me MCP Deployment${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "Environment: ${YELLOW}$ENVIRONMENT${NC}"
echo -e "Inventory: ${YELLOW}$INVENTORY_FILE${NC}"
if [ -n "$TAGS" ]; then
    echo -e "Tags: ${YELLOW}$TAGS${NC}"
fi
if [ -n "$CHECK_MODE" ]; then
    echo -e "Mode: ${YELLOW}CHECK (dry run)${NC}"
fi
echo -e "${GREEN}======================================${NC}"
echo ""

# Confirm deployment
if [ -z "$CHECK_MODE" ] && [ "$ENVIRONMENT" == "production" ]; then
    read -p "Are you sure you want to deploy to PRODUCTION? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Deployment cancelled${NC}"
        exit 0
    fi
fi

# Run deployment
echo -e "${YELLOW}Starting deployment...${NC}"
echo -e "${YELLOW}Running: $ANSIBLE_CMD${NC}"
echo ""

# Execute the command
$ANSIBLE_CMD

# Check if deployment was successful
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}Deployment completed successfully!${NC}"
    echo -e "${GREEN}======================================${NC}"
    
    if [ "$ENVIRONMENT" == "production" ]; then
        echo -e "Service URL: ${YELLOW}https://api.remembermcp.com${NC}"
    else
        echo -e "Service URL: ${YELLOW}https://staging.remembermcp.com${NC}"
    fi
    echo -e "Health Check: ${YELLOW}https://$ENVIRONMENT.remembermcp.com/health${NC}"
    echo -e "API Docs: ${YELLOW}https://$ENVIRONMENT.remembermcp.com/swagger${NC}"
else
    echo ""
    echo -e "${RED}======================================${NC}"
    echo -e "${RED}Deployment failed!${NC}"
    echo -e "${RED}======================================${NC}"
    exit 1
fi