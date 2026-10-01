---
name: ROSA Command Authoring
description: "Add or edit Cobra commands in openshift/rosa while keeping command wiring thin and package logic aligned with repo structure."
---

# ROSA Command Authoring

Use this skill for new or changed commands, flags, help text, or command flow.

1. Read [AGENTS.md](../../../AGENTS.md) and its command, architecture, and
   error guides. Inspect the nearest comparable command implementation.
2. Apply the command guide to wiring, entrypoint, output, and prompt behavior.
3. Check the command tree and flag contracts named in the command guide.
4. Follow the guide's documentation generation rule when help text changes.
5. Select and run the relevant checks from [CONTRIBUTING.md](../../../CONTRIBUTING.md).
