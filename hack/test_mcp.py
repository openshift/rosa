#!/usr/bin/env python3
#
# Test script for ROSA MCP server using stdio transport
# Queries the MCP server for all available tools and resources
#

import json
import subprocess
import sys
import time
import select
import os
import signal
from pathlib import Path

# Colors for output
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'  # No Color

def print_colored(color, message):
    print(f"{color}{message}{Colors.NC}")

def find_repo_root():
    """Find the repository root by looking for go.mod"""
    current = Path(__file__).resolve().parent
    while current != current.parent:
        if (current / 'go.mod').exists():
            return str(current)
        current = current.parent
    raise RuntimeError("Could not find repo root (go.mod)")

def check_go_command():
    """Check if go command is available"""
    try:
        result = subprocess.run(['go', 'version'],
                               capture_output=True, timeout=5)
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False

def send_mcp_message(proc, message, expected_id, timeout=5):
    """Send a JSON-RPC message and read the response"""
    # Send message to stdin (with newline)
    proc.stdin.write((json.dumps(message) + '\n').encode())
    proc.stdin.flush()

    # Read response from stdout
    end_time = time.time() + timeout
    buffer = ""

    while time.time() < end_time:
        # Check if process is still running
        if proc.poll() is not None:
            return None, f"Process exited with code {proc.returncode}"

        # Try to read from stdout (non-blocking)
        if proc.stdout in select.select([proc.stdout], [], [], 0.1)[0]:
            chunk = proc.stdout.read(1)
            if chunk:
                buffer += chunk.decode('utf-8', errors='ignore')

                # Check if we have a complete line
                if '\n' in buffer:
                    lines = buffer.split('\n')
                    # Process complete lines
                    for line in lines[:-1]:
                        line = line.strip()
                        if not line:
                            continue

                        try:
                            msg = json.loads(line)
                            msg_id = msg.get('id')
                            if msg_id == expected_id:
                                return msg, None
                        except json.JSONDecodeError:
                            continue

                    # Keep incomplete line in buffer
                    buffer = lines[-1]

    return None, "Timeout waiting for response"

def call_mcp_tool(proc, tool_name, arguments=None, timeout=10):
    """Call an MCP tool and return the result"""
    if arguments is None:
        arguments = {}

    call_msg = {
        "jsonrpc": "2.0",
        "id": 100,  # Use high ID to avoid conflicts
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": arguments
        }
    }

    response, error = send_mcp_message(proc, call_msg, 100, timeout)
    if error:
        return None, error

    if response and 'error' in response:
        return None, f"Tool call failed: {json.dumps(response['error'])}"

    return response, None

def main():
    # Check if go is available
    if not check_go_command():
        print_colored(Colors.RED, "Error: go command not found")
        sys.exit(1)

    # Find repo root
    try:
        repo_root = find_repo_root()
    except RuntimeError as e:
        print_colored(Colors.RED, f"Error: {e}")
        sys.exit(1)

    print_colored(Colors.BLUE, f"Starting MCP server with stdio transport (from {repo_root})...")

    # Start rosa mcp serve process using go run
    try:
        proc = subprocess.Popen(
            ['go', 'run', '-mod=mod', './cmd/rosa', 'mcp', 'serve'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            cwd=repo_root,
            bufsize=0  # Unbuffered
        )
    except Exception as e:
        print_colored(Colors.RED, f"Failed to start server: {e}")
        sys.exit(1)

    # Cleanup function
    def cleanup():
        print_colored(Colors.BLUE, "\nStopping server...")
        proc.terminate()
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait()

    # Register cleanup on exit
    def signal_handler(sig, frame):
        cleanup()
        sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Give server a moment to start
    time.sleep(0.3)

    # Check if server started successfully
    time.sleep(0.2)  # Give it a moment
    if proc.poll() is not None:
        print_colored(Colors.RED, "Server failed to start")
        # Try to read stderr (non-blocking)
        stderr_lines = []
        while True:
            if proc.stderr in select.select([proc.stderr], [], [], 0.1)[0]:
                line = proc.stderr.readline()
                if line:
                    stderr_lines.append(line.decode('utf-8', errors='ignore'))
                else:
                    break
            else:
                break
        if stderr_lines:
            print(''.join(stderr_lines))
        sys.exit(1)

    try:
        # Initialize connection
        print_colored(Colors.BLUE, "\n=== Initializing MCP connection ===")
        init_msg = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "mcp-test-client",
                    "version": "1.0.0"
                }
            }
        }

        init_response, error = send_mcp_message(proc, init_msg, 1, timeout=10)
        if error:
            print_colored(Colors.RED, f"Failed to get initialize response: {error}")
            sys.exit(1)

        if init_response and 'error' in init_response:
            print_colored(Colors.RED, "Initialize failed:")
            print(json.dumps(init_response['error'], indent=2))
            sys.exit(1)

        print_colored(Colors.GREEN, "✓ Initialized")

        # Send initialized notification
        notif_msg = {
            "jsonrpc": "2.0",
            "method": "notifications/initialized"
        }
        proc.stdin.write((json.dumps(notif_msg) + '\n').encode())
        proc.stdin.flush()
        time.sleep(0.1)

        # Query tools
        print_colored(Colors.BLUE, "\n=== Querying Tools ===")
        tools_msg = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list",
            "params": {}
        }

        tools_response, error = send_mcp_message(proc, tools_msg, 2, timeout=10)
        if error:
            print_colored(Colors.RED, f"Failed to get tools/list response: {error}")
        elif tools_response and 'error' in tools_response:
            print_colored(Colors.RED, "Tools/list failed:")
            print(json.dumps(tools_response['error'], indent=2))
        else:
            tools = tools_response.get('result', {}).get('tools', [])
            print_colored(Colors.GREEN, f"✓ Found {len(tools)} tools\n")
            for tool in tools:
                name = tool.get('name', 'Unknown')
                desc = tool.get('description', 'No description')
                print(f"  • {name}: {desc}")

        # Query resources
        print_colored(Colors.BLUE, "\n=== Querying Resources ===")
        resources_msg = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "resources/list",
            "params": {}
        }

        resources_response, error = send_mcp_message(proc, resources_msg, 3, timeout=10)
        if error:
            print_colored(Colors.RED, f"Failed to get resources/list response: {error}")
        elif resources_response and 'error' in resources_response:
            print_colored(Colors.RED, "Resources/list failed:")
            print(json.dumps(resources_response['error'], indent=2))
        else:
            resources = resources_response.get('result', {}).get('resources', [])
            print_colored(Colors.GREEN, f"✓ Found {len(resources)} resources\n")
            for resource in resources:
                uri = resource.get('uri', 'Unknown')
                name = resource.get('name', 'No name')
                desc = resource.get('description', 'No description')
                print(f"  • {uri}: {name} - {desc}")

        # Call rosa_whoami tool
        print_colored(Colors.BLUE, "\n=== Calling rosa_whoami Tool ===")
        whoami_response, error = call_mcp_tool(proc, "rosa_whoami", {}, timeout=15)
        if error:
            print_colored(Colors.RED, f"Failed to call rosa_whoami: {error}")
        else:
            result = whoami_response.get('result', {})
            if result.get('isError'):
                print_colored(Colors.RED, "rosa_whoami returned an error:")
                content = result.get('content', [])
                for item in content:
                    if item.get('type') == 'text':
                        print(item.get('text', ''))
            else:
                print_colored(Colors.GREEN, "✓ rosa_whoami executed successfully\n")
                content = result.get('content', [])
                for item in content:
                    if item.get('type') == 'text':
                        print(item.get('text', ''))

        # Read rosa://clusters resource
        print_colored(Colors.BLUE, "\n=== Reading rosa://clusters Resource ===")
        read_resource_msg = {
            "jsonrpc": "2.0",
            "id": 4,
            "method": "resources/read",
            "params": {
                "uri": "rosa://clusters"
            }
        }

        resource_response, error = send_mcp_message(proc, read_resource_msg, 4, timeout=15)
        if error:
            print_colored(Colors.RED, f"Failed to read rosa://clusters resource: {error}")
        elif resource_response and 'error' in resource_response:
            print_colored(Colors.RED, "resources/read failed:")
            print(json.dumps(resource_response['error'], indent=2))
        else:
            result = resource_response.get('result', {})
            contents = result.get('contents', [])
            print_colored(Colors.GREEN, f"✓ Successfully read rosa://clusters resource ({len(contents)} content items)\n")
            for content_item in contents:
                uri = content_item.get('uri', 'Unknown')
                mime_type = content_item.get('mimeType', 'Unknown')
                text = content_item.get('text', '')
                print(f"  URI: {uri}")
                print(f"  MIME Type: {mime_type}")
                if text:
                    # Try to parse as JSON for pretty printing
                    try:
                        data = json.loads(text)
                        print(f"  Content (formatted):")
                        print(json.dumps(data, indent=2))
                    except json.JSONDecodeError:
                        print(f"  Content:\n{text}")
                print()

        print_colored(Colors.GREEN, "\n=== Done ===")

    finally:
        cleanup()

if __name__ == '__main__':
    main()
