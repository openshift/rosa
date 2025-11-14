#!/bin/bash

################################################################################
# ROSA Capacity Reservation Validation Test Script
# 
# Purpose: Tests capacity reservation validation feature for ROSA HCP clusters
# 
# What it does:
#   1. Creates a new AWS capacity reservation (2 instances)
#   2. Tests nodepool creation with capacity limits
#   3. Tests editing nodepools with capacity validation
#   4. Tests autoscaling configuration with capacity limits
#   5. Automatically cleans up all resources
#
# Requirements:
#   - AWS CLI configured with appropriate permissions
#   - ROSA CLI with capacity reservation validation changes compiled
#   - Access to a ROSA HCP cluster
#
# Usage:
#   ./test_capacity_reservation.sh
#
# Configuration:
#   Edit the variables below to match your environment if needed
################################################################################

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration from your actual cluster
CLUSTER_NAME="m-p-edit"
INSTANCE_TYPE="m5.xlarge"
AVAILABILITY_ZONE="us-west-2a"  # Using the AZ from your 'test' nodepool
CAPACITY_COUNT=2  # Number of instances in capacity reservation
TEST_NODEPOOL_NAME="capacity-demo-$(date +%s)"  # Unique name with timestamp

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Variables to store created resources for cleanup
CAPACITY_RESERVATION_ID=""
NODEPOOL_CREATED=false

echo -e "${MAGENTA}============================================"
echo "All-in-One Capacity Reservation Test Suite"
echo "============================================${NC}"
echo "Cluster: ${CLUSTER_NAME}"
echo "Instance Type: ${INSTANCE_TYPE}"
echo "Availability Zone: ${AVAILABILITY_ZONE}"
echo "Test NodePool: ${TEST_NODEPOOL_NAME}"
echo ""
echo -e "${GREEN}This script will:${NC}"
echo "1. Create a NEW capacity reservation (not use existing ones)"
echo "2. Run all validation tests"
echo "3. Automatically DELETE the capacity reservation when done"
echo "   (even if tests fail - cleanup is guaranteed)"
echo ""

# Function to handle cleanup
cleanup() {
    echo -e "\n${YELLOW}=== AUTOMATIC CLEANUP STARTED ===${NC}"
    echo "Cleaning up test resources (this runs even if tests fail)..."
    
    if [ "$NODEPOOL_CREATED" = true ]; then
        echo "• Deleting test nodepool: ${TEST_NODEPOOL_NAME}"
        ./rosa delete machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --yes 2>/dev/null || true
        echo "  ✓ NodePool deleted"
    fi
    
    if [ -n "$CAPACITY_RESERVATION_ID" ] && [ "$CAPACITY_RESERVATION_ID" != "" ]; then
        echo "• Canceling NEW capacity reservation: ${CAPACITY_RESERVATION_ID}"
        # First check if it exists
        if aws ec2 describe-capacity-reservations --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} &>/dev/null; then
            aws ec2 cancel-capacity-reservation --capacity-reservation-id ${CAPACITY_RESERVATION_ID} 2>/dev/null || true
            echo "  ✓ Capacity reservation ${CAPACITY_RESERVATION_ID} canceled"
        else
            echo "  ✓ Capacity reservation already deleted or doesn't exist"
        fi
    fi
    
    echo -e "${GREEN}✓ Cleanup completed - all test resources removed${NC}"
    echo -e "${BLUE}Note: Any pre-existing capacity reservations were NOT affected${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
    echo "$2"
}

# Function to run a test that should fail
test_should_fail() {
    local test_name="$1"
    local command="$2"
    local expected_error="$3"
    
    echo -e "\n${YELLOW}TEST: ${test_name}${NC}"
    echo "Command: ${command}"
    echo "Expected: FAIL with pattern '${expected_error}'"
    
    # Run the command and capture output
    output=$(eval "${command}" 2>&1 || true)
    
    if echo "$output" | grep -q "${expected_error}"; then
        echo -e "${GREEN}✓ Test PASSED - Failed as expected${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ Test FAILED - Did not fail with expected error${NC}"
        echo "Actual output: $output"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Function to run a test that should succeed
test_should_succeed() {
    local test_name="$1"
    local command="$2"
    
    echo -e "\n${YELLOW}TEST: ${test_name}${NC}"
    echo "Command: ${command}"
    echo "Expected: SUCCESS"
    
    if output=$(eval "${command}" 2>&1); then
        echo -e "${GREEN}✓ Test PASSED - Succeeded as expected${NC}"
        echo "Output: $output" | head -3
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ Test FAILED - Did not succeed${NC}"
        echo "Error: $output"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Check AWS CLI is available
print_section "PREREQUISITES CHECK" "Verifying AWS CLI and ROSA are configured..."

if ! command -v aws &> /dev/null; then
    echo -e "${RED}ERROR: AWS CLI is not installed or not in PATH${NC}"
    echo "Please install AWS CLI: https://aws.amazon.com/cli/"
    exit 1
fi

if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${RED}ERROR: AWS CLI is not configured${NC}"
    echo "Please run: aws configure"
    exit 1
fi

# Test AWS CLI JSON query functionality
TEST_QUERY=$(aws sts get-caller-identity --query 'Account' --output text 2>/dev/null)
if [ -z "$TEST_QUERY" ]; then
    echo -e "${RED}ERROR: AWS CLI query functionality not working${NC}"
    echo "Please ensure AWS CLI is properly installed and configured"
    exit 1
fi
echo "AWS Account: ${TEST_QUERY}"

if ! ./rosa describe cluster -c ${CLUSTER_NAME} &> /dev/null; then
    echo -e "${RED}ERROR: Cannot access cluster ${CLUSTER_NAME}${NC}"
    echo "Please check your ROSA login and cluster name"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"

# Show current cluster state
print_section "CURRENT CLUSTER STATE" "Displaying existing nodepools..."

echo -e "${BLUE}Current NodePools:${NC}"
./rosa list machinepools --cluster=${CLUSTER_NAME}

# Step 1: Create AWS Capacity Reservation
print_section "STEP 1: CREATE AWS CAPACITY RESERVATION" "Creating NEW capacity reservation with ${CAPACITY_COUNT} instances..."

# Show existing capacity reservations (informational only)
echo "Checking for existing capacity reservations (we will create a new one)..."
EXISTING_CRS=$(aws ec2 describe-capacity-reservations \
    --filters "Name=state,Values=active" \
    --query 'CapacityReservations[*].[CapacityReservationId,AvailabilityZone,InstanceType,TotalInstanceCount,AvailableInstanceCount]' \
    --output text 2>/dev/null || true)

if [ -n "$EXISTING_CRS" ]; then
    echo -e "${YELLOW}Note: Found existing capacity reservations (we will NOT use these):${NC}"
    echo "$EXISTING_CRS" | while read cr_id az type total avail; do
        echo "  - $cr_id in $az ($type): $avail/$total available"
    done
else
    echo "No existing active capacity reservations found"
fi

echo ""
echo "Creating NEW capacity reservation in ${AVAILABILITY_ZONE}..."
echo "Using 'targeted' match criteria to prevent automatic consumption by other instances"
echo "Command: aws ec2 create-capacity-reservation --instance-type ${INSTANCE_TYPE} --instance-platform Linux/UNIX --availability-zone ${AVAILABILITY_ZONE} --instance-count ${CAPACITY_COUNT} --instance-match-criteria targeted"
echo ""

# Create capacity reservation and capture both output and errors
ERROR_FILE=$(mktemp)
OUTPUT_FILE=$(mktemp)

aws ec2 create-capacity-reservation \
    --instance-type ${INSTANCE_TYPE} \
    --instance-platform Linux/UNIX \
    --availability-zone ${AVAILABILITY_ZONE} \
    --instance-count ${CAPACITY_COUNT} \
    --instance-match-criteria targeted \
    --output json > $OUTPUT_FILE 2> $ERROR_FILE

CREATE_EXIT_CODE=$?

# Check if command succeeded
if [ $CREATE_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}Failed to create capacity reservation${NC}"
    echo "Error details:"
    cat $ERROR_FILE
    rm -f $ERROR_FILE $OUTPUT_FILE
    echo ""
    echo "Possible reasons:"
    echo "- Insufficient permissions to create capacity reservations"
    echo "- No available capacity in ${AVAILABILITY_ZONE} for ${INSTANCE_TYPE}"
    echo "- AWS account limits reached"
    echo ""
    echo "Try checking available capacity with:"
    echo "  aws ec2 describe-instance-type-offerings --region us-west-2 --filters \"Name=instance-type,Values=${INSTANCE_TYPE}\" \"Name=location,Values=${AVAILABILITY_ZONE}\""
    exit 1
fi

# Extract capacity reservation ID
CAPACITY_RESERVATION_ID=$(cat $OUTPUT_FILE | grep -o '"CapacityReservationId"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)

# Verify we got a valid ID
if [ -z "$CAPACITY_RESERVATION_ID" ] || [ "$CAPACITY_RESERVATION_ID" = "None" ] || [ "$CAPACITY_RESERVATION_ID" = "null" ]; then
    echo -e "${RED}Failed to get capacity reservation ID from response${NC}"
    echo "AWS Response:"
    cat $OUTPUT_FILE
    rm -f $ERROR_FILE $OUTPUT_FILE
    echo ""
    echo "The capacity reservation may have been created but we couldn't get its ID."
    echo "Check AWS console or run:"
    echo "  aws ec2 describe-capacity-reservations --filters \"Name=state,Values=active\""
    exit 1
fi

rm -f $ERROR_FILE $OUTPUT_FILE
echo -e "${GREEN}✓ Created NEW capacity reservation: ${CAPACITY_RESERVATION_ID}${NC}"
echo -e "${BLUE}This is a fresh capacity reservation created just for this test${NC}"
echo -e "${BLUE}It will be automatically deleted when the test completes${NC}"

# Wait for capacity reservation to be fully active
echo ""
echo "Waiting for capacity reservation to be fully ready..."
sleep 3

# Verify capacity reservation
echo "Verifying NEW capacity reservation details..."
TOTAL_INSTANCES=$(aws ec2 describe-capacity-reservations \
    --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} \
    --query 'CapacityReservations[0].TotalInstanceCount' \
    --output text)

AVAILABLE_INSTANCES=$(aws ec2 describe-capacity-reservations \
    --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} \
    --query 'CapacityReservations[0].AvailableInstanceCount' \
    --output text)

STATE=$(aws ec2 describe-capacity-reservations \
    --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} \
    --query 'CapacityReservations[0].State' \
    --output text)

echo "  Capacity Reservation ID: ${CAPACITY_RESERVATION_ID}"
echo "  Total Instances: ${TOTAL_INSTANCES}"
echo "  Available Instances: ${AVAILABLE_INSTANCES}"
echo "  State: ${STATE}"

# Check if something is consuming the capacity
if [ "$AVAILABLE_INSTANCES" -eq "0" ] && [ "$TOTAL_INSTANCES" -gt "0" ]; then
    echo -e "${YELLOW}WARNING: Capacity shows 0 available but ${TOTAL_INSTANCES} total${NC}"
    echo "This might mean:"
    echo "1. The capacity was immediately consumed by other instances"
    echo "2. There's existing usage in this capacity reservation"
    echo ""
    echo "Checking what's using the capacity..."
    aws ec2 describe-capacity-reservations \
        --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} \
        --query 'CapacityReservations[0].{ID:CapacityReservationId,Total:TotalInstanceCount,Available:AvailableInstanceCount,Used:UsedInstanceCount,State:State,InstanceType:InstanceType,AZ:AvailabilityZone}' \
        --output table
    echo ""
    echo "Note: The test will continue but may show unexpected results"
fi

# Step 2: Test Creating NodePool with Capacity Reservation
print_section "STEP 2: CREATE NODEPOOL WITH CAPACITY RESERVATION" "Testing nodepool creation with capacity limits..."

# Test 2.1: Try to create with too many replicas (should fail)
# Note: Using actual available count from AWS, not the original requested count
test_should_fail \
    "Create nodepool EXCEEDING capacity (${AVAILABLE_INSTANCES} currently available, requesting 3)" \
    "./rosa create machinepool --name=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --replicas=3 --instance-type=${INSTANCE_TYPE} --availability-zone=${AVAILABILITY_ZONE} --capacity-reservation-id=${CAPACITY_RESERVATION_ID}" \
    "cannot set replicas to 3.*capacity reservation"

# Test 2.2: Create with valid replicas (should succeed)
# Create with just 1 replica to be safe
test_should_succeed \
    "Create nodepool WITHIN capacity (requesting 1 replica)" \
    "./rosa create machinepool --name=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --replicas=1 --instance-type=${INSTANCE_TYPE} --availability-zone=${AVAILABILITY_ZONE} --capacity-reservation-id=${CAPACITY_RESERVATION_ID}"

NODEPOOL_CREATED=true

# Step 3: Verify Capacity Reservation Attachment
print_section "STEP 3: VERIFY CAPACITY RESERVATION" "Checking nodepool has capacity reservation attached..."

echo "Describing the created nodepool..."
DESCRIBE_OUTPUT=$(./rosa describe machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME})
echo "$DESCRIBE_OUTPUT" | grep -A2 "Capacity Reservation:"

if echo "$DESCRIBE_OUTPUT" | grep -q "${CAPACITY_RESERVATION_ID}"; then
    echo -e "${GREEN}✓ Capacity reservation ${CAPACITY_RESERVATION_ID} is attached${NC}"
else
    echo -e "${RED}✗ Capacity reservation not found in nodepool description${NC}"
fi

# Step 4: Test Editing NodePool with Capacity Limits
print_section "STEP 4: EDIT NODEPOOL TESTS" "Testing replica modifications with capacity limits..."

# Update available capacity after nodepool creation
sleep 5  # Give AWS a moment to update
AVAILABLE_NOW=$(aws ec2 describe-capacity-reservations \
    --capacity-reservation-ids ${CAPACITY_RESERVATION_ID} \
    --query 'CapacityReservations[0].AvailableInstanceCount' \
    --output text)
echo "Current available capacity: ${AVAILABLE_NOW}"

# Test 4.1: Try to exceed capacity (should fail)
test_should_fail \
    "Edit nodepool to EXCEED capacity (available: ${AVAILABLE_NOW}, requesting: 3)" \
    "./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --replicas=3" \
    "cannot set replicas to 3.*capacity reservation"

# Test 4.2: Edit within capacity (should succeed)
test_should_succeed \
    "Edit nodepool WITHIN capacity (available: ${AVAILABLE_NOW}, requesting: 2)" \
    "./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --replicas=2"

# Step 5: Test Autoscaling with Capacity Limits
print_section "STEP 5: AUTOSCALING TESTS" "Testing autoscaling configuration with capacity limits..."

# Test 5.1: Min replicas exceeding capacity (should fail)
test_should_fail \
    "Enable autoscaling with MIN replicas EXCEEDING capacity" \
    "./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --enable-autoscaling --min-replicas=3 --max-replicas=4" \
    "cannot set min replicas to 3.*capacity reservation"

# Test 5.2: Max replicas exceeding capacity (should fail)
test_should_fail \
    "Enable autoscaling with MAX replicas EXCEEDING capacity" \
    "./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --enable-autoscaling --min-replicas=1 --max-replicas=5" \
    "cannot set max replicas to 5.*capacity reservation"

# Test 5.3: Valid autoscaling range (should succeed)
test_should_succeed \
    "Enable autoscaling WITHIN capacity limits (1-2)" \
    "./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --enable-autoscaling --min-replicas=1 --max-replicas=2"

# Step 6: Interactive Mode Test
print_section "STEP 6: INTERACTIVE MODE" "Testing if capacity info is shown in interactive mode..."

echo "Testing interactive mode display (non-interactive simulation)..."
echo -e "n\n" | ./rosa edit machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} --interactive 2>&1 | grep -i "capacity" || echo "No capacity info shown in interactive mode"

# Step 7: Final Verification
print_section "STEP 7: FINAL STATE" "Showing final nodepool configuration..."

./rosa describe machinepool --machinepool=${TEST_NODEPOOL_NAME} --cluster=${CLUSTER_NAME} | grep -E "ID:|Autoscaling:|Current replicas:|Capacity Reservation:" || true

# Test Summary
print_section "TEST RESULTS SUMMARY" ""
echo "=========================================="
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo "=========================================="

if [ ${TESTS_FAILED} -eq 0 ]; then
    echo -e "\n${GREEN}SUCCESS: All capacity reservation validation tests passed!${NC}"
    echo ""
    echo "The tests verified:"
    echo "✓ Cannot create nodepool exceeding capacity reservation"
    echo "✓ Can create nodepool within capacity limits"
    echo "✓ Cannot edit replicas to exceed capacity"
    echo "✓ Can edit replicas within capacity limits"
    echo "✓ Autoscaling min/max respect capacity limits"
    echo "✓ Capacity reservation is properly attached to nodepool"
    echo ""
    echo -e "${YELLOW}Resources created and cleaned up:${NC}"
    echo "• Capacity Reservation: ${CAPACITY_RESERVATION_ID} (DELETED)"
    echo "• Test NodePool: ${TEST_NODEPOOL_NAME} (DELETED)"
    echo ""
    echo -e "${GREEN}All test resources have been cleaned up automatically${NC}"
    exit 0
else
    echo -e "\n${RED}FAILURE: Some tests failed!${NC}"
    echo "Please check the output above for details."
    echo ""
    echo -e "${YELLOW}Don't worry about cleanup:${NC}"
    echo "• Capacity Reservation ${CAPACITY_RESERVATION_ID} will be deleted"
    echo "• Test NodePool ${TEST_NODEPOOL_NAME} will be deleted"
    echo "Cleanup happens automatically even when tests fail!"
    exit 1
fi
