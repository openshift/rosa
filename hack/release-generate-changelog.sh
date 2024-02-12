#!/bin/bash

# Check to ensure we have the `jira` CLI installed and accessible from the $PATH
if ! [ -x "$(command -v jira)" ]; then
  echo "Error: Please install the 'jira' CLI and ensure it is on your \$PATH"
  exit 1
fi

# Validate input parameters
if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <release-label> <previous-release>"
  exit 1
fi

# Assign input parameters to variables
release_label="$1"
previous_release="$2"

# Create a temporary file for the awk code
awk_script=$(mktemp)

# Write the awk code to the temporary file
cat > "$awk_script" << 'AWK_SCRIPT'
#!/usr/bin/awk -f

NR > 1 {
  types[NR-1] = $1;
  keys[NR-1] = $2;
  $1 = $2 = "";
  gsub(/^[ \t]+/, "", $0);
  split($0, arr, "\t");
  summaries[NR-1] = arr[1];
  statuses[NR-1] = arr[2];
  labels[NR-1] = arr[3];
}
END {
  # Print each issue's information with separators "---"
  for (i = 1; i <= NR - 2; i++) {
    print "Type: " types[i];
    print "Key: " keys[i];
    print "Summary: " summaries[i];
    print "---";
  }


  printf "\n"
  printf "---------------\n"
  printf "To Be Used in Errata\n"
  # Print all keys as a comma-separated string
  printf "All Keys: %s\n", join(keys, ",");
  printf "---------------\n\n"
}


# Function to join array elements with a delimiter
function join(arr, delimiter,    result) {
  result = arr[1];
  for (i = 2; i <= length(arr); i++) {
    result = result delimiter arr[i];
  }
  return result;
}
AWK_SCRIPT

# Run the jira issue list command and store the output in a variable
jira_output=$(jira issue list --jql "project in (\"Openshift Cluster Manager\", \"Service Development A\") AND labels = $release_label" --plain --columns TYPE,KEY,SUMMARY,STATUS,LABELS)

# print awk output to help build the list of keys for errata
awk -f "$awk_script" <<< "$jira_output"

# Use awk to parse the output and execute the script from the temporary file
keys=$(awk -f "$awk_script" <<< "$jira_output" | awk '/^Key:/{print $2}')
commit_messages=$(git log "$previous_release"..HEAD --pretty=format:"%s" --no-merges --reverse)

# Filter and print only the commit messages that contain a key from the list of keys
while read -r line; do
  for key in $keys; do
    if echo "$line" | grep -q "$key"; then
      # Remove Jira ticket ID and pipe character from the commit message
      filtered_message=$(echo "$line" | sed -E 's/^[[:alnum:]]+-[0-9]+ \| //')
      echo "$filtered_message"
      break
    fi
  done
done <<< "$commit_messages"

# Remove the temporary awk script file
rm "$awk_script"