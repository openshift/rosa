#!/bin/bash
#
# Copyright (c) 2021 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This lists the jira cards addressed in the current release

previous=$(git tag --list --sort version:refname | tail -n2 | head -n1)
current="v$(cat pkg/info/info.go | grep -o --color=never '[0-9.]*' | tail -n1)"

# Find JIRA references in commit log
jiras=$(git log $previous...$current | grep -Eo --color=never '\<[A-Z]{1,10}-[0-9]+\>' | sort | uniq)

# Show simple list of JIRA IDs
echo $jiras

# Show list of JIRA links
echo "$jiras" | sed 's|^|https://issues.redhat.com/browse/|'
