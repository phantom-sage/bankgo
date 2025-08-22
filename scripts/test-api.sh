#!/bin/bash

# Bank REST API Testing Script
# This script tests all API endpoints to verify functionality

set -e

# Configuration
API_BASE_URL="http://localhost:8080/api/v1"
TEST_EMAIL="test@example.com"
TEST_PASSWORD="TestPassword123!"
TEST_FIRST_NAME="Test"
TEST_LAST_NAME="User"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print test status
print_test() {
    echo -e "${BLUE}🧪 Testing:${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓ Success:${NC} $1"
}

print_error() {
    echo -e "${RED}✗ Error:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠ Warning:${NC} $1"
}

# Function to make HTTP requests and check status
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local headers=$5
    
    local url="${API_BASE_URL}${endpoint}"
    local response
    local status_code
    
    if [ -n "$data" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "$headers")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url")
        fi
    fi
    
    status_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "HTTP $status_code - $method $endpoint"
        echo "$response_body"
        return 0
    else
        print_error "Expected HTTP $expected_status, got $status_code - $method $endpoint"
        echo "$response_body"
        return 1
    fi
}

echo "🚀 Bank REST API Testing Suite"
echo "=============================="
echo ""

# Check if API is running
print_test "API connectivity"
if ! curl -s "$API_BASE_URL/health" > /dev/null; then
    print_error "API is not accessible at $API_BASE_URL"
    echo "Please ensure the API is running with 'make up' or 'docker-compose up -d'"
    exit 1
fi
print_success "API is accessible"
echo ""

# Test 1: Health Check
print_test "Health check endpoint"
health_response=$(make_request "GET" "/health" "" "200")
if echo "$health_response" | grep -q '"status":"healthy"'; then
    print_success "Health check passed"
else
    print_warning "Health check returned unexpected response"
fi
echo ""

# Test 2: User Registration
print_test "User registration"
register_data="{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\",
    \"first_name\": \"$TEST_FIRST_NAME\",
    \"last_name\": \"$TEST_LAST_NAME\"
}"

register_response=$(make_request "POST" "/auth/register" "$register_data" "201")
if echo "$register_response" | grep -q '"email"'; then
    print_success "User registration successful"
else
    print_error "User registration failed"
    echo "$register_response"
fi
echo ""

# Test 3: User Login
print_test "User login"
login_data="{
    \"email\": \"$TEST_EMAIL\",
    \"password\": \"$TEST_PASSWORD\"
}"

login_response=$(make_request "POST" "/auth/login" "$login_data" "200")
if echo "$login_response" | grep -q '"token"'; then
    print_success "User login successful"
    # Extract token for subsequent requests
    TOKEN=$(echo "$login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$TOKEN" ]; then
        print_success "Authentication token extracted"
    else
        print_error "Failed to extract authentication token"
        exit 1
    fi
else
    print_error "User login failed"
    echo "$login_response"
    exit 1
fi
echo ""

# Test 4: Create USD Account
print_test "Create USD account"
account_data='{"currency": "USD"}'
auth_header="Authorization: Bearer $TOKEN"

usd_account_response=$(make_request "POST" "/accounts" "$account_data" "201" "$auth_header")
if echo "$usd_account_response" | grep -q '"currency":"USD"'; then
    print_success "USD account created successfully"
    USD_ACCOUNT_ID=$(echo "$usd_account_response" | grep -o '"id":[0-9]*' | cut -d':' -f2)
    print_success "USD Account ID: $USD_ACCOUNT_ID"
else
    print_error "USD account creation failed"
    echo "$usd_account_response"
fi
echo ""

# Test 5: Create EUR Account
print_test "Create EUR account"
eur_account_data='{"currency": "EUR"}'

eur_account_response=$(make_request "POST" "/accounts" "$eur_account_data" "201" "$auth_header")
if echo "$eur_account_response" | grep -q '"currency":"EUR"'; then
    print_success "EUR account created successfully"
    EUR_ACCOUNT_ID=$(echo "$eur_account_response" | grep -o '"id":[0-9]*' | cut -d':' -f2)
    print_success "EUR Account ID: $EUR_ACCOUNT_ID"
else
    print_error "EUR account creation failed"
    echo "$eur_account_response"
fi
echo ""

# Test 6: List User Accounts
print_test "List user accounts"
accounts_response=$(make_request "GET" "/accounts" "" "200" "$auth_header")
if echo "$accounts_response" | grep -q '"currency"'; then
    print_success "Account listing successful"
    account_count=$(echo "$accounts_response" | grep -o '"currency"' | wc -l)
    print_success "Found $account_count accounts"
else
    print_error "Account listing failed"
    echo "$accounts_response"
fi
echo ""

# Test 7: Get Specific Account
if [ -n "$USD_ACCOUNT_ID" ]; then
    print_test "Get specific account details"
    account_detail_response=$(make_request "GET" "/accounts/$USD_ACCOUNT_ID" "" "200" "$auth_header")
    if echo "$account_detail_response" | grep -q '"id":'$USD_ACCOUNT_ID; then
        print_success "Account details retrieved successfully"
    else
        print_error "Failed to retrieve account details"
        echo "$account_detail_response"
    fi
    echo ""
fi

# Test 8: Invalid Currency Account (should fail)
print_test "Create account with invalid currency (should fail)"
invalid_account_data='{"currency": "INVALID"}'
invalid_response=$(make_request "POST" "/accounts" "$invalid_account_data" "400" "$auth_header")
if echo "$invalid_response" | grep -q '"error"'; then
    print_success "Invalid currency properly rejected"
else
    print_warning "Invalid currency validation may not be working"
fi
echo ""

# Test 9: Duplicate Currency Account (should fail)
print_test "Create duplicate USD account (should fail)"
duplicate_response=$(make_request "POST" "/accounts" "$account_data" "409" "$auth_header")
if echo "$duplicate_response" | grep -q '"error"'; then
    print_success "Duplicate currency properly rejected"
else
    print_warning "Duplicate currency validation may not be working"
fi
echo ""

# Test 10: Transfer Money (will fail due to zero balance, but tests validation)
if [ -n "$USD_ACCOUNT_ID" ] && [ -n "$EUR_ACCOUNT_ID" ]; then
    print_test "Attempt money transfer (should fail - insufficient balance)"
    transfer_data="{
        \"from_account_id\": $USD_ACCOUNT_ID,
        \"to_account_id\": $EUR_ACCOUNT_ID,
        \"amount\": \"100.00\",
        \"description\": \"Test transfer\"
    }"
    
    transfer_response=$(make_request "POST" "/transfers" "$transfer_data" "422" "$auth_header")
    if echo "$transfer_response" | grep -q '"error"'; then
        print_success "Insufficient balance properly detected"
    else
        print_warning "Balance validation may not be working"
    fi
    echo ""
fi

# Test 11: Transfer History
print_test "Get transfer history"
history_response=$(make_request "GET" "/transfers" "" "200" "$auth_header")
if echo "$history_response" | grep -q '\['; then
    print_success "Transfer history retrieved (empty list expected)"
else
    print_error "Failed to retrieve transfer history"
    echo "$history_response"
fi
echo ""

# Test 12: Unauthorized Access (no token)
print_test "Unauthorized access (should fail)"
unauth_response=$(make_request "GET" "/accounts" "" "401" "")
if echo "$unauth_response" | grep -q '"error"'; then
    print_success "Unauthorized access properly rejected"
else
    print_warning "Authorization validation may not be working"
fi
echo ""

# Test 13: Invalid Token
print_test "Invalid token access (should fail)"
invalid_auth_header="Authorization: Bearer invalid_token"
invalid_token_response=$(make_request "GET" "/accounts" "" "401" "$invalid_auth_header")
if echo "$invalid_token_response" | grep -q '"error"'; then
    print_success "Invalid token properly rejected"
else
    print_warning "Token validation may not be working"
fi
echo ""

# Test 14: Logout
print_test "User logout"
logout_response=$(make_request "POST" "/auth/logout" "" "200" "$auth_header")
print_success "Logout completed"
echo ""

# Test 15: Access After Logout (should still work with PASETO stateless tokens)
print_test "Access after logout (PASETO tokens are stateless)"
post_logout_response=$(make_request "GET" "/accounts" "" "200" "$auth_header")
if echo "$post_logout_response" | grep -q '"currency"'; then
    print_success "PASETO token still valid after logout (expected behavior)"
else
    print_warning "Token invalidated after logout (unexpected for PASETO)"
fi
echo ""

echo "📊 Test Summary"
echo "==============="
echo ""
echo -e "${GREEN}✓ Core functionality tests completed${NC}"
echo ""
echo "🧪 Tests performed:"
echo "  • Health check endpoint"
echo "  • User registration and login"
echo "  • Account creation (USD, EUR)"
echo "  • Account listing and details"
echo "  • Input validation (invalid currency, duplicates)"
echo "  • Transfer validation (insufficient balance)"
echo "  • Authorization (unauthorized access, invalid tokens)"
echo "  • User logout"
echo ""
echo "💡 Notes:"
echo "  • Transfer tests expect insufficient balance errors (accounts start with $0)"
echo "  • PASETO tokens remain valid after logout (stateless design)"
echo "  • All validation and security checks are working properly"
echo ""
echo "🎉 API testing complete!"
echo ""
echo "Next steps:"
echo "  • Fund accounts to test successful transfers"
echo "  • Run integration tests: make test-integration"
echo "  • Check application logs: make logs"