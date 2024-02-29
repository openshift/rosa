#!/bin/bash

# Check to ensure we have the `jira` CLI installed and accessible from the $PATH
if ! [ -x "$(command -v jira)" ]; then
  echo "Error: Please install the 'jira' CLI and ensure it is on your \$PATH"
  exit 1
fi

# Check if the command-line argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <release_version_branch>"
  exit 1
fi

release_version="$1"
commit_output=$(git log "$release_version"..HEAD --oneline --no-merges --format="%s" --no-decorate --reverse | tr '[:upper:]' '[:lower:]')

# Regular expression pattern to extract Jira ticket numbers and commit messages
pattern="^([^|]+)\|(.+)$"

# Array to store Jira ticket numbers
jira_tickets=()

while IFS= read -r line; do
  if [[ $line =~ $pattern ]]; then
    ticket="${BASH_REMATCH[1]}"
    jira_tickets+=("$ticket")
  fi
done <<< "$commit_output"

# Create a comma-separated list of Jira tickets
jira_list=$(IFS=, ; echo "${jira_tickets[*]}")

# Create the JQL query for the list of Jira tickets
jql="project = \"Openshift Cluster Manager\" AND issue in ($jira_list)"

# Create the jira issue list command
jira_command="jira issue list --jql '$jql'"

# Execute the Jira command
eval "$jira_command"