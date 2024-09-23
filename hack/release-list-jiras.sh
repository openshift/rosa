#!/bin/bash

# Check to ensure we have the `jira` CLI installed and accessible from the $PATH
if ! [ -x "$(command -v jira)" ]; then
  echo "Error: Please install the 'jira' CLI and ensure it is on your \$PATH"
  exit 1
fi

# Validate input parameters
if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <current-release> <previous-release>"
  exit 1
fi

# Assign input parameters to variables
current_release="rosa_cli_$1"
previous_release="release_$2"

commit_output=$(git log "$previous_release"..HEAD --oneline --no-merges --format="%s" --no-decorate --reverse | tr '[:upper:]' '[:lower:]')

# Regular expression pattern to extract Jira ticket numbers and commit messages
pattern="^(revert[[:space:]]*\")?([^|]+)\|(.+)$"

# Array to store Jira ticket numbers
jira_tickets=()

while IFS= read -r line; do
  if [[ $line =~ $pattern && ! ${BASH_REMATCH[1]} ]]; then
    ticket="${BASH_REMATCH[2]}"
    jira_tickets+=("$ticket")
  fi
done <<< "$commit_output"


# Create a comma-separated list of Jira tickets
jira_list=$(IFS=, ; echo "${jira_tickets[*]}")

# Create a space-separated list of capitalized Jira tickets
errata_list=$(IFS=' ' ; echo "${jira_tickets[@]^^}")
echo -e "List of JIRA's to be used in Errata \n$errata_list"

# Create the JQL query for the list of Jira tickets
jql="project = \"Openshift Cluster Manager\" AND issue in ($jira_list) AND labels not in (no-qe) AND (fixVersion is EMPTY OR fixVersion = $current_release)"


# Create the jira issue list command
jira_command="jira issue list --jql '$jql' --columns key,assignee,status,summary"

# Execute the Jira command
eval "$jira_command"