#!/bin/bash

# Note: To run this test script, use:
# curl -X POST "http://localhost:8080/api/test/generate-users?count=5" -H "Content-Type: application/json" && ./backend/test_endpoints.sh

# Base URL
BASE_URL="http://localhost:8080"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Database connection details
DB_NAME="matcherator"
DB_USER="postgres"
DB_PASS="postgres"
DB_HOST="localhost"
DB_PORT="5432"

# Create a test image file
echo "Creating test image file..."
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" | base64 -d > test_image.png

# Function to make API calls and check response
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    local token=$5
    local file=$6
    
    echo -e "\n${GREEN}=== Testing: $description ===${NC}"
    echo -e "${GREEN}Endpoint: $method $endpoint${NC}"
    
    if [ -n "$file" ]; then
        if [ -n "$token" ]; then
            response=$(curl -s -X $method "$BASE_URL$endpoint" \
                -H "Authorization: Bearer $token" \
                -F "file=@$file")
        else
            response=$(curl -s -X $method "$BASE_URL$endpoint" \
                -F "file=@$file")
        fi
    elif [ -n "$data" ]; then
        if [ -n "$token" ]; then
            response=$(curl -s -X $method "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $token" \
                -d "$data")
        else
            response=$(curl -s -X $method "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ -n "$token" ]; then
            response=$(curl -s -X $method "$BASE_URL$endpoint" \
                -H "Authorization: Bearer $token")
        else
            response=$(curl -s -X $method "$BASE_URL$endpoint")
        fi
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Request successful${NC}"
        echo -e "\n${GREEN}Response:${NC}"
        # Try to pretty print if it's JSON
        if echo "$response" | jq . >/dev/null 2>&1; then
            echo "$response" | jq .
        else
            echo "$response"
        fi
        echo -e "\n${GREEN}----------------------------------------${NC}"
        echo "$response" # Return just the body for further processing if needed
    else
        echo -e "${RED}✗ Request failed${NC}"
        echo -e "\n${RED}Error response:${NC}"
        echo "$response"
        echo -e "\n${RED}----------------------------------------${NC}"
        echo ""
    fi
}

# Generate test data using the proper endpoint
echo "Generating test data..."
curl -X POST "http://localhost:8080/api/test/generate-users?count=5" -H "Content-Type: application/json"

# Get test users directly from database
echo "Getting test users from database..."
PROVIDER_EMAIL=$(PGPASSWORD=$DB_PASS psql -U $DB_USER -h $DB_HOST -d $DB_NAME -c "SELECT email FROM users WHERE role = 'provider' ORDER BY id LIMIT 1;" -t -A)
RECIPIENT_EMAIL=$(PGPASSWORD=$DB_PASS psql -U $DB_USER -h $DB_HOST -d $DB_NAME -c "SELECT email FROM users WHERE role = 'recipient' ORDER BY id LIMIT 1;" -t -A)

echo "Provider email: $PROVIDER_EMAIL"
echo "Recipient email: $RECIPIENT_EMAIL"

# Get user IDs
PROVIDER_ID=$(PGPASSWORD=$DB_PASS psql -U $DB_USER -h $DB_HOST -d $DB_NAME -c "SELECT id FROM users WHERE email = '$PROVIDER_EMAIL';" -t -A)
RECIPIENT_ID=$(PGPASSWORD=$DB_PASS psql -U $DB_USER -h $DB_HOST -d $DB_NAME -c "SELECT id FROM users WHERE email = '$RECIPIENT_EMAIL';" -t -A)

echo "Provider ID: $PROVIDER_ID"
echo "Recipient ID: $RECIPIENT_ID"

# Login to get tokens
echo "Logging in to get tokens..."
PROVIDER_TOKEN=$(curl -s -X POST "http://localhost:8080/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$PROVIDER_EMAIL\", \"password\": \"testpass123\"}" | jq -r '.token')

RECIPIENT_TOKEN=$(curl -s -X POST "http://localhost:8080/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$RECIPIENT_EMAIL\", \"password\": \"testpass123\"}" | jq -r '.token')

echo "Provider token: $PROVIDER_TOKEN"
echo "Recipient token: $RECIPIENT_TOKEN"

# Test endpoints as provider
echo -e "\n${GREEN}Testing endpoints as provider...${NC}"

# Test basic profile update
test_endpoint "PUT" "/api/me/profile" '{
  "organization_name": "Test Provider Org",
  "mission_statement": "Test mission statement",
  "sectors": ["Technology"],
  "target_groups": ["Youth"]
}' "Update Provider Profile" "$PROVIDER_TOKEN"

# Test endpoints as provider
test_endpoint "GET" "/api/me" "" "Get Provider Basic Info" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/me/profile" "" "Get Provider Profile" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/users/$RECIPIENT_ID" "" "Get User" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/users/$RECIPIENT_ID/full" "" "Get Full User" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/users/$RECIPIENT_ID/profile" "" "Get User Profile" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/users/$RECIPIENT_ID/bio" "" "Get User Bio" "$PROVIDER_TOKEN"

# Test additional endpoints
echo -e "\n${GREEN}Testing additional endpoints...${NC}"
test_endpoint "GET" "/api/users" "" "Get All Users" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/me/bio" "" "Get My Bio" "$PROVIDER_TOKEN"

# Test potential matches
echo -e "\n${GREEN}Testing potential matches...${NC}"
test_endpoint "GET" "/api/potential-matches" "" "Get Potential Matches" "$PROVIDER_TOKEN"

# Test WebSocket endpoints (these will only test the HTTP upgrade request)
echo -e "\n${GREEN}Testing WebSocket endpoints...${NC}"
test_endpoint "GET" "/ws/chat/$CHAT_ID" "" "WebSocket Chat Connection" "$PROVIDER_TOKEN"
test_endpoint "GET" "/ws/notifications" "" "WebSocket Notifications Connection" "$PROVIDER_TOKEN"

# Test connections
echo -e "\n${GREEN}Testing connections...${NC}"

# First, delete any existing connections
test_endpoint "DELETE" "/api/connections/$RECIPIENT_ID" "" "Delete existing connection" "$PROVIDER_TOKEN"

# Create new connection
test_endpoint "POST" "/api/connections" "{\"target_id\": $RECIPIENT_ID}" "Create Connection" "$PROVIDER_TOKEN"

# Get connections
test_endpoint "GET" "/api/connections" "" "Get Connections" "$PROVIDER_TOKEN"

# Test notifications
echo -e "\n${GREEN}Testing notifications...${NC}"
test_endpoint "GET" "/api/notifications" "" "Get Notifications" "$PROVIDER_TOKEN"
test_endpoint "POST" "/api/notifications/read" "" "Mark Notifications as Read" "$PROVIDER_TOKEN"

# Test chat setup
echo -e "\n${GREEN}Testing chat setup...${NC}"

# First enable chat for both users
echo "Enabling chat for provider..."
test_endpoint "PUT" "/api/chat/preferences" '{"opt_in": true}' "Enable chat for provider" "$PROVIDER_TOKEN"

echo "Enabling chat for recipient..."
test_endpoint "PUT" "/api/chat/preferences" '{"opt_in": true}' "Enable chat for recipient" "$RECIPIENT_TOKEN"

# Verify chat preferences
echo "Verifying chat preferences for provider..."
test_endpoint "GET" "/api/chat/preferences" "" "Get Provider Chat Preferences" "$PROVIDER_TOKEN"

echo "Verifying chat preferences for recipient..."
test_endpoint "GET" "/api/chat/preferences" "" "Get Recipient Chat Preferences" "$RECIPIENT_TOKEN"

# Get list of chats
echo "Getting chats for provider..."
test_endpoint "GET" "/api/chat" "" "Get Provider Chats" "$PROVIDER_TOKEN"

echo "Getting chats for recipient..."
test_endpoint "GET" "/api/chat" "" "Get Recipient Chats" "$RECIPIENT_TOKEN"

# Get chat ID from provider's chats
CHAT_ID=$(curl -s -X GET "$BASE_URL/api/chat" -H "Authorization: Bearer $PROVIDER_TOKEN" | jq -r '.[0].id')

# Test chat messages
echo -e "\n${GREEN}Testing chat messages...${NC}"
test_endpoint "GET" "/api/chat/$CHAT_ID/messages" "" "Get Chat Messages" "$PROVIDER_TOKEN"
test_endpoint "POST" "/api/chat/$CHAT_ID/messages/read" "" "Mark Messages as Read" "$PROVIDER_TOKEN"

# Test WebSocket connection (this is informational only, as we can't test WebSocket in a shell script)
echo -e "\n${GREEN}Note: WebSocket chat functionality must be tested separately using a WebSocket client${NC}"

# Test status
echo -e "\n${GREEN}Testing status...${NC}"
test_endpoint "GET" "/api/status/1" "" "Get Status" "$PROVIDER_TOKEN"
test_endpoint "GET" "/api/status" "" "Get My Status" "$PROVIDER_TOKEN"

# Test upload endpoints with actual file
echo -e "\n${GREEN}Testing profile picture upload/delete...${NC}"
test_endpoint "POST" "/api/upload/profile-picture" "" "Upload Profile Picture" "$PROVIDER_TOKEN" "test_image.png"
test_endpoint "DELETE" "/api/upload/profile-picture" "" "Delete Profile Picture" "$PROVIDER_TOKEN"

# Test endpoints as recipient
echo -e "\n${GREEN}Testing endpoints as recipient...${NC}"

# Test basic profile update
test_endpoint "PUT" "/api/me/profile" '{
  "organization_name": "Test Recipient Org",
  "mission_statement": "Test mission statement",
  "sectors": ["Education"],
  "target_groups": ["Students"]
}' "Update Recipient Profile" "$RECIPIENT_TOKEN"

test_endpoint "GET" "/api/me" "" "Get Recipient Basic Info" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/me/profile" "" "Get Recipient Profile" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/users/$PROVIDER_ID" "" "Get User" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/users/$PROVIDER_ID/full" "" "Get Full User" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/users/$PROVIDER_ID/profile" "" "Get User Profile" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/users/$PROVIDER_ID/bio" "" "Get User Bio" "$RECIPIENT_TOKEN"

# Test connections as recipient
echo -e "\n${GREEN}Testing connections as recipient...${NC}"
test_endpoint "GET" "/api/connections" "" "Get Connections" "$RECIPIENT_TOKEN"

# Test notifications as recipient
echo -e "\n${GREEN}Testing notifications as recipient...${NC}"
test_endpoint "GET" "/api/notifications" "" "Get Notifications" "$RECIPIENT_TOKEN"
test_endpoint "POST" "/api/notifications/read" "" "Mark Notifications as Read" "$RECIPIENT_TOKEN"

# Test status as recipient
echo -e "\n${GREEN}Testing status as recipient...${NC}"
test_endpoint "GET" "/api/status/2" "" "Get Status" "$RECIPIENT_TOKEN"
test_endpoint "GET" "/api/status" "" "Get My Status" "$RECIPIENT_TOKEN"

# Test upload endpoints with actual file
echo -e "\n${GREEN}Testing profile picture upload/delete for recipient...${NC}"
test_endpoint "POST" "/api/upload/profile-picture" "" "Upload Profile Picture" "$RECIPIENT_TOKEN" "test_image.png"
test_endpoint "DELETE" "/api/upload/profile-picture" "" "Delete Profile Picture" "$RECIPIENT_TOKEN"

# Clean up test image
rm test_image.png

echo -e "\n${GREEN}All tests completed${NC}" 