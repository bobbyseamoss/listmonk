#!/bin/bash

# Deployment Management Helper Script
# Manage both listmonk420 and listmonk-comma deployments

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_menu() {
    clear
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         Listmonk Deployment Management                        ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${GREEN}Select Deployment:${NC}"
    echo -e "  ${YELLOW}1.${NC} Original (listmonk420 / bobbyseamoss.com)"
    echo -e "  ${YELLOW}2.${NC} COMMA (listmonk-comma / enjoycomma.com)"
    echo -e "  ${YELLOW}3.${NC} Compare Deployments"
    echo -e "  ${YELLOW}4.${NC} Check Both Deployments Status"
    echo -e "  ${YELLOW}q.${NC} Quit"
    echo ""
    echo -n "Choice: "
}

show_actions() {
    local deployment=$1
    echo ""
    echo -e "${GREEN}Select Action for ${deployment}:${NC}"
    echo -e "  ${YELLOW}1.${NC} Deploy"
    echo -e "  ${YELLOW}2.${NC} View Logs"
    echo -e "  ${YELLOW}3.${NC} Check Status"
    echo -e "  ${YELLOW}4.${NC} Access Database"
    echo -e "  ${YELLOW}5.${NC} Health Check"
    echo -e "  ${YELLOW}6.${NC} List Revisions"
    echo -e "  ${YELLOW}7.${NC} Restart Container"
    echo -e "  ${YELLOW}b.${NC} Back to Main Menu"
    echo ""
    echo -n "Choice: "
}

# Original deployment functions
original_deploy() {
    echo -e "${GREEN}Deploying to listmonk420...${NC}"
    ./deploy.sh
}

original_logs() {
    echo -e "${GREEN}Viewing logs for listmonk420...${NC}"
    az containerapp logs show \
        --name listmonk420 \
        --resource-group rg-listmonk420 \
        --follow
}

original_status() {
    echo -e "${GREEN}Checking status for listmonk420...${NC}"
    az containerapp show \
        --name listmonk420 \
        --resource-group rg-listmonk420 \
        --query "{status:properties.runningStatus, health:properties.healthState, replicas:properties.runningReplicaCount}" \
        -o table
}

original_db() {
    if [ -z "$LISTMONK_DB_PASSWORD" ]; then
        echo -e "${RED}Error: LISTMONK_DB_PASSWORD not set${NC}"
        return 1
    fi
    PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
        -h listmonk420-db.postgres.database.azure.com \
        -U listmonkadmin \
        -d listmonk
}

original_health() {
    echo -e "${GREEN}Checking health for listmonk420...${NC}"
    curl -s https://list.bobbyseamoss.com/api/health | jq .
}

original_revisions() {
    echo -e "${GREEN}Listing revisions for listmonk420...${NC}"
    az containerapp revision list \
        --name listmonk420 \
        --resource-group rg-listmonk420 \
        --query "[].{name:name, active:properties.active, created:properties.createdTime, replicas:properties.replicas}" \
        -o table
}

original_restart() {
    echo -e "${GREEN}Restarting listmonk420...${NC}"
    LATEST_REVISION=$(az containerapp show \
        --name listmonk420 \
        --resource-group rg-listmonk420 \
        --query "properties.latestRevisionName" -o tsv)
    az containerapp revision restart \
        --resource-group rg-listmonk420 \
        --name listmonk420 \
        --revision "$LATEST_REVISION"
}

# COMMA deployment functions
comma_deploy() {
    echo -e "${GREEN}Deploying to listmonk-comma...${NC}"
    ./deploy-comma.sh
}

comma_logs() {
    echo -e "${GREEN}Viewing logs for listmonk-comma...${NC}"
    az containerapp logs show \
        --name listmonk-comma \
        --resource-group comma-rg \
        --follow
}

comma_status() {
    echo -e "${GREEN}Checking status for listmonk-comma...${NC}"
    az containerapp show \
        --name listmonk-comma \
        --resource-group comma-rg \
        --query "{status:properties.runningStatus, health:properties.healthState, replicas:properties.runningReplicaCount}" \
        -o table
}

comma_db() {
    if [ -z "$LISTMONK_COMMA_DB_PASSWORD" ]; then
        echo -e "${RED}Error: LISTMONK_COMMA_DB_PASSWORD not set${NC}"
        return 1
    fi
    PGPASSWORD="$LISTMONK_COMMA_DB_PASSWORD" psql \
        -h listmonk420-db.postgres.database.azure.com \
        -U listmonkadmin \
        -d listmonk_comma
}

comma_health() {
    echo -e "${GREEN}Checking health for listmonk-comma...${NC}"
    curl -s https://list.enjoycomma.com/api/health | jq .
}

comma_revisions() {
    echo -e "${GREEN}Listing revisions for listmonk-comma...${NC}"
    az containerapp revision list \
        --name listmonk-comma \
        --resource-group comma-rg \
        --query "[].{name:name, active:properties.active, created:properties.createdTime, replicas:properties.replicas}" \
        -o table
}

comma_restart() {
    echo -e "${GREEN}Restarting listmonk-comma...${NC}"
    LATEST_REVISION=$(az containerapp show \
        --name listmonk-comma \
        --resource-group comma-rg \
        --query "properties.latestRevisionName" -o tsv)
    az containerapp revision restart \
        --resource-group comma-rg \
        --name listmonk-comma \
        --revision "$LATEST_REVISION"
}

compare_deployments() {
    clear
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║              Deployment Comparison                             ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    printf "%-30s %-35s %-35s\n" "Resource" "Original (listmonk420)" "COMMA (listmonk-comma)"
    echo "────────────────────────────────────────────────────────────────────────────────────────────────"
    printf "%-30s %-35s %-35s\n" "Resource Group" "rg-listmonk420" "comma-rg"
    printf "%-30s %-35s %-35s\n" "Container App" "listmonk420" "listmonk-comma"
    printf "%-30s %-35s %-35s\n" "Container Registry" "listmonk420acr" "listmonkcommaacr"
    printf "%-30s %-35s %-35s\n" "Database Server" "listmonk420-db" "${GREEN}listmonk420-db (shared)${NC}"
    printf "%-30s %-35s %-35s\n" "Database Name" "listmonk" "listmonk_comma"
    printf "%-30s %-35s %-35s\n" "Domain" "list.bobbyseamoss.com" "list.enjoycomma.com"
    printf "%-30s %-35s %-35s\n" "Deploy Script" "./deploy.sh" "./deploy-comma.sh"
    printf "%-30s %-35s %-35s\n" "Env Var Prefix" "LISTMONK_DB_*" "LISTMONK_COMMA_DB_*"
    echo ""
    echo -e "${YELLOW}Note:${NC} Both deployments share the PostgreSQL server but use separate databases."
    echo -e "      Saves approximately ${GREEN}\$25-35/month${NC} compared to separate servers."
    echo ""
    read -p "Press Enter to continue..."
}

check_both_status() {
    clear
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║              Deployment Status Summary                         ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    echo -e "${YELLOW}=== Original Deployment (listmonk420) ===${NC}"
    if az containerapp show --name listmonk420 --resource-group rg-listmonk420 > /dev/null 2>&1; then
        STATUS=$(az containerapp show \
            --name listmonk420 \
            --resource-group rg-listmonk420 \
            --query "properties.runningStatus" -o tsv)
        HEALTH=$(az containerapp show \
            --name listmonk420 \
            --resource-group rg-listmonk420 \
            --query "properties.healthState" -o tsv)
        REPLICAS=$(az containerapp show \
            --name listmonk420 \
            --resource-group rg-listmonk420 \
            --query "properties.runningReplicaCount" -o tsv)

        echo -e "Status: ${GREEN}${STATUS}${NC}"
        echo -e "Health: ${GREEN}${HEALTH}${NC}"
        echo -e "Replicas: ${GREEN}${REPLICAS}${NC}"
        echo -e "URL: ${BLUE}https://list.bobbyseamoss.com${NC}"

        # Health check
        if curl -s -f https://list.bobbyseamoss.com/api/health > /dev/null 2>&1; then
            echo -e "Health Check: ${GREEN}✓ Passing${NC}"
        else
            echo -e "Health Check: ${RED}✗ Failing${NC}"
        fi
    else
        echo -e "${RED}Not deployed${NC}"
    fi

    echo ""
    echo -e "${YELLOW}=== COMMA Deployment (listmonk-comma) ===${NC}"
    if az containerapp show --name listmonk-comma --resource-group comma-rg > /dev/null 2>&1; then
        STATUS=$(az containerapp show \
            --name listmonk-comma \
            --resource-group comma-rg \
            --query "properties.runningStatus" -o tsv)
        HEALTH=$(az containerapp show \
            --name listmonk-comma \
            --resource-group comma-rg \
            --query "properties.healthState" -o tsv)
        REPLICAS=$(az containerapp show \
            --name listmonk-comma \
            --resource-group comma-rg \
            --query "properties.runningReplicaCount" -o tsv)

        echo -e "Status: ${GREEN}${STATUS}${NC}"
        echo -e "Health: ${GREEN}${HEALTH}${NC}"
        echo -e "Replicas: ${GREEN}${REPLICAS}${NC}"
        echo -e "URL: ${BLUE}https://list.enjoycomma.com${NC}"

        # Health check
        if curl -s -f https://list.enjoycomma.com/api/health > /dev/null 2>&1; then
            echo -e "Health Check: ${GREEN}✓ Passing${NC}"
        else
            echo -e "Health Check: ${RED}✗ Failing${NC}"
        fi
    else
        echo -e "${RED}Not deployed${NC}"
    fi

    echo ""
    read -p "Press Enter to continue..."
}

handle_original() {
    while true; do
        show_actions "Original (listmonk420)"
        read choice
        case $choice in
            1) original_deploy ;;
            2) original_logs ;;
            3) original_status ;;
            4) original_db ;;
            5) original_health ;;
            6) original_revisions ;;
            7) original_restart ;;
            b) break ;;
            *) echo -e "${RED}Invalid choice${NC}" ;;
        esac
        if [ "$choice" != "b" ]; then
            echo ""
            read -p "Press Enter to continue..."
        fi
    done
}

handle_comma() {
    while true; do
        show_actions "COMMA (listmonk-comma)"
        read choice
        case $choice in
            1) comma_deploy ;;
            2) comma_logs ;;
            3) comma_status ;;
            4) comma_db ;;
            5) comma_health ;;
            6) comma_revisions ;;
            7) comma_restart ;;
            b) break ;;
            *) echo -e "${RED}Invalid choice${NC}" ;;
        esac
        if [ "$choice" != "b" ]; then
            echo ""
            read -p "Press Enter to continue..."
        fi
    done
}

# Main loop
while true; do
    show_menu
    read choice
    case $choice in
        1) handle_original ;;
        2) handle_comma ;;
        3) compare_deployments ;;
        4) check_both_status ;;
        q|Q) echo -e "${GREEN}Goodbye!${NC}"; exit 0 ;;
        *) echo -e "${RED}Invalid choice${NC}"; sleep 1 ;;
    esac
done
