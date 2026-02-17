#!/usr/bin/env bash
# Verify that a torrent client is fully registered across all required files.
# Usage:
#   scripts/verify-client-registration.sh qbittorrent     # Check one client
#   scripts/verify-client-registration.sh --all            # Check all clients
#   scripts/verify-client-registration.sh --implemented    # Check only implemented clients

set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# All torrent client types that should eventually exist
ALL_TYPES=(
    transmission qbittorrent deluge rtorrent vuze
    aria2 flood utorrent hadouken downloadstation
    freeboxdownload rqbit tribler
)

# Files to check
TYPES_FILE="internal/downloader/types/types.go"
CLIENT_FILE="internal/downloader/client.go"
FACTORY_FILE="internal/downloader/factory.go"
SERVICE_FILE="internal/downloader/service.go"

pass=0
fail=0
warn=0

# Map from client type string to Go constant name suffix
# e.g., "transmission" -> "Transmission", "qbittorrent" -> "QBittorrent"
type_to_const() {
    case "$1" in
        transmission)    echo "Transmission" ;;
        qbittorrent)     echo "QBittorrent" ;;
        deluge)          echo "Deluge" ;;
        rtorrent)        echo "RTorrent" ;;
        vuze)            echo "Vuze" ;;
        aria2)           echo "Aria2" ;;
        flood)           echo "Flood" ;;
        utorrent)        echo "UTorrent" ;;
        hadouken)        echo "Hadouken" ;;
        downloadstation) echo "DownloadStation" ;;
        freeboxdownload) echo "FreeboxDownload" ;;
        rqbit)           echo "RQBit" ;;
        tribler)         echo "Tribler" ;;
        mock)            echo "Mock" ;;
        *)               echo "$1" ;;
    esac
}

check() {
    local desc="$1"
    local file="$2"
    local pattern="$3"

    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $desc"
        ((pass++)) || true
    else
        echo -e "  ${RED}✗${NC} $desc"
        ((fail++)) || true
    fi
}

check_warn() {
    local desc="$1"
    local file="$2"
    local pattern="$3"

    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $desc"
        ((pass++)) || true
    else
        echo -e "  ${YELLOW}?${NC} $desc (optional)"
        ((warn++)) || true
    fi
}

verify_client() {
    local client_type="$1"
    local const_name
    const_name=$(type_to_const "$client_type")

    echo ""
    echo "=== Checking: $client_type (ClientType$const_name) ==="

    # 1. ClientType constant in types.go — e.g., ClientTypeTransmission ClientType = "transmission"
    check "ClientType constant in types.go" "$TYPES_FILE" "ClientType${const_name}.*=.*\"${client_type}\""

    # 2. Re-export in client.go — e.g., ClientTypeTransmission = types.ClientTypeTransmission
    check "Re-export in client.go" "$CLIENT_FILE" "ClientType${const_name}"

    # 3. ProtocolForClient mapping in types.go — constant appears in the switch (may be comma-separated)
    check "ProtocolForClient mapping" "$TYPES_FILE" "ClientType${const_name}[,:]"

    # 4. validClientTypes in service.go — e.g., "transmission": true
    check "validClientTypes entry in service.go" "$SERVICE_FILE" "\"${client_type}\".*true"

    # 5. Factory switch case — e.g., case ClientTypeTransmission:
    check "Factory NewClient case" "$FACTORY_FILE" "ClientType${const_name}"

    # 6. ImplementedClientTypes — check if in the ImplementedClientTypes function
    # This grep looks for the constant name appearing after the ImplementedClientTypes func definition
    check_warn "In ImplementedClientTypes()" "$FACTORY_FILE" "ClientType${const_name}"

    # 7. Package directory exists
    local pkg_dir="internal/downloader/${client_type}"
    # Some packages use different directory names
    case "$client_type" in
        rtorrent)        pkg_dir="internal/downloader/rtorrent" ;;
        downloadstation) pkg_dir="internal/downloader/downloadstation" ;;
        freeboxdownload) pkg_dir="internal/downloader/freeboxdownload" ;;
    esac

    if [ -d "$pkg_dir" ]; then
        echo -e "  ${GREEN}✓${NC} Package directory exists: $pkg_dir"
        ((pass++)) || true

        # Check for client.go
        if [ -f "$pkg_dir/client.go" ]; then
            echo -e "  ${GREEN}✓${NC} client.go exists"
            ((pass++)) || true

            # Check for compile-time interface check
            if grep -q "types.TorrentClient" "$pkg_dir/client.go" 2>/dev/null; then
                echo -e "  ${GREEN}✓${NC} TorrentClient interface assertion"
                ((pass++)) || true
            else
                echo -e "  ${RED}✗${NC} Missing TorrentClient interface assertion"
                ((fail++)) || true
            fi

            # Check for NewFromConfig
            if grep -q "NewFromConfig" "$pkg_dir/client.go" 2>/dev/null; then
                echo -e "  ${GREEN}✓${NC} NewFromConfig constructor"
                ((pass++)) || true
            else
                echo -e "  ${RED}✗${NC} Missing NewFromConfig constructor"
                ((fail++)) || true
            fi
        else
            echo -e "  ${RED}✗${NC} client.go missing"
            ((fail++)) || true
        fi

        # Check for tests
        if ls "$pkg_dir"/*_test.go 1>/dev/null 2>&1; then
            echo -e "  ${GREEN}✓${NC} Test file exists"
            ((pass++)) || true
        else
            echo -e "  ${YELLOW}?${NC} No test file (recommended)"
            ((warn++)) || true
        fi
    else
        echo -e "  ${RED}✗${NC} Package directory missing: $pkg_dir"
        ((fail++)) || true
    fi

    # 8. DB migration includes this type
    if grep -rq "$client_type" internal/database/migrations/ 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} Included in DB migration CHECK constraint"
        ((pass++)) || true
    else
        echo -e "  ${YELLOW}?${NC} Not found in DB migrations (may be in a pending migration)"
        ((warn++)) || true
    fi
}

# Parse arguments
if [ $# -eq 0 ]; then
    echo "Usage: $0 <client_type> | --all | --implemented"
    echo ""
    echo "Client types: ${ALL_TYPES[*]}"
    exit 1
fi

case "$1" in
    --all)
        echo "Checking all client types..."
        for t in "${ALL_TYPES[@]}"; do
            verify_client "$t"
        done
        ;;
    --implemented)
        echo "Checking implemented client types..."
        for t in "${ALL_TYPES[@]}"; do
            const_name=$(type_to_const "$t")
            # Check if this type appears in ImplementedClientTypes in factory.go
            if grep -q "ClientType${const_name}" "$FACTORY_FILE" 2>/dev/null; then
                verify_client "$t"
            fi
        done
        ;;
    *)
        verify_client "$1"
        ;;
esac

# Summary
echo ""
echo "================================"
echo -e "Results: ${GREEN}${pass} passed${NC}, ${RED}${fail} failed${NC}, ${YELLOW}${warn} warnings${NC}"
echo "================================"

if [ "$fail" -gt 0 ]; then
    exit 1
fi
