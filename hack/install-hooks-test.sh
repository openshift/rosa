#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0


set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
export GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL="$test_root/global-config"
touch "$GIT_CONFIG_GLOBAL"
mkdir -p "$test_root/bin"
cat > "$test_root/bin/go" <<'STUB'
#!/bin/sh
printf '%s\n' /unused
STUB
cat > "$test_root/bin/pre-commit" <<'STUB'
#!/bin/sh
printf '%s\n' "$*" >> "$HOOK_TEST_LOG"
exit "${HOOK_TEST_EXIT:-0}"
STUB
chmod +x "$test_root/bin/go" "$test_root/bin/pre-commit"
export PATH="$test_root/bin:$PATH"

new_fixture() {
	fixture="$test_root/$1"
	mkdir -p "$fixture/.bingo"
	cp "$repo_root/.bingo/Variables.mk" "$fixture/.bingo/Variables.mk"
	git init -q "$fixture"
	export HOOK_TEST_LOG="$fixture/installer.log"
	output="$fixture/output.log"
	: > "$GIT_CONFIG_GLOBAL"
}

install_hooks() {
	make --no-print-directory -C "$fixture" -f "$repo_root/Makefile" install-hooks > "$output" 2>&1
}

assert_installed() {
	grep -Fxq 'install --install-hooks --hook-type pre-commit --hook-type commit-msg --hook-type pre-push' "$HOOK_TEST_LOG"
	grep -q '^Installed pre-commit hooks' "$output"
}

assert_skipped() {
	test ! -e "$HOOK_TEST_LOG"
	grep -q '^Hook installation skipped:' "$output"
	grep -q 'All checks remain mandatory.*CONTRIBUTING.md' "$output"
	if grep -q '^Installed pre-commit hooks' "$output"; then
		return 1
	fi
}

new_fixture normal
install_hooks
assert_installed
echo 'PASS: normal installation includes every configured stage'

new_fixture legacy
git -C "$fixture" config --local core.hooksPath .githooks
install_hooks
assert_installed
if git -C "$fixture" config --local --get core.hooksPath; then
	exit 1
fi
echo 'PASS: local legacy migration'

new_fixture inherited
git config --global core.hooksPath /custom/hooks
cp "$GIT_CONFIG_GLOBAL" "$fixture/global-before"
install_hooks
assert_skipped
cmp "$GIT_CONFIG_GLOBAL" "$fixture/global-before"
test "$(git -C "$fixture" config --get core.hooksPath)" = /custom/hooks
echo 'PASS: inherited custom path preserved without invoking installer'

new_fixture legacy_with_inherited
git config --global core.hooksPath /custom/hooks
git -C "$fixture" config --local core.hooksPath .githooks
cp "$GIT_CONFIG_GLOBAL" "$fixture/global-before"
install_hooks
assert_skipped
cmp "$GIT_CONFIG_GLOBAL" "$fixture/global-before"
test "$(git -C "$fixture" config --get core.hooksPath)" = /custom/hooks
echo 'PASS: legacy migration preserves inherited custom path'

new_fixture local_custom
git -C "$fixture" config --local core.hooksPath custom-hooks
install_hooks
assert_skipped
test "$(git -C "$fixture" config --local --get core.hooksPath)" = custom-hooks
echo 'PASS: local custom path preserved'

new_fixture installer_failure
if HOOK_TEST_EXIT=23 install_hooks; then
	exit 1
fi
test -s "$HOOK_TEST_LOG"
if grep -q '^Installed pre-commit hooks' "$output"; then
	exit 1
fi
echo 'PASS: genuine installer failure remains an error'

new_fixture legacy_installer_failure
git -C "$fixture" config --local core.hooksPath .githooks
if HOOK_TEST_EXIT=23 install_hooks; then
	exit 1
fi
test "$(git -C "$fixture" config --local --get core.hooksPath)" = .githooks
echo 'PASS: installer failure restores local legacy hooks'

new_fixture legacy_missing_installer
git -C "$fixture" config --local core.hooksPath .githooks
mkdir -p "$test_root/no-installer"
for executable in make git sed; do
	ln -s "$(command -v "$executable")" "$test_root/no-installer/$executable"
done
cp "$test_root/bin/go" "$test_root/no-installer/go"
if PATH="$test_root/no-installer" install_hooks; then
	exit 1
fi
test ! -e "$HOOK_TEST_LOG"
test "$(git -C "$fixture" config --local --get core.hooksPath)" = .githooks
grep -q 'preserving legacy hooks' "$output"
echo 'PASS: missing installer preserves local legacy hooks'
