#!/usr/bin/env bash

# PULL_BASE_SHA and PULL_PULL_SHA are set in prow job
PULL_BASE_SHA="${PULL_BASE_SHA:-$(git merge-base master HEAD)}"
PULL_PULL_SHA="${PULL_PULL_SHA:-HEAD}"

pattern="^[A-Z]+-[0-9]+ \| (feat|fix|docs|style|refactor|test|chore|build|ci|perf): .*$"

commits=$(git rev-list --no-merges $PULL_BASE_SHA..$PULL_PULL_SHA)

for sha in $commits; do
  message=$(git log --format=%B -n 1 $sha)
  if ! [[ $message =~ $pattern ]]; then
    echo "Invalid commit message: $message"
    echo "Expected format: JIRA_TICKET | TYPE: MESSAGE"
    echo "Please check CONTRIBUTING.md for more details"
    exit 1
  fi
done

echo "All commit message are valid"
