#!/usr/bin/env python3
"""ROSA HyperFleet v1 E2E Parity Scoreboard.

Source of truth: Ginkgo labels in tests/e2e/ (via ginkgo run --dry-run).
- hyperfleet-validated : passing tests
- hyperfleet-sanity    : standalone lifecycle spec
- hyperfleet-deferred  : known gaps / blocked upstream
- hyperfleet-na        : not applicable (classic/OCM/harness)
- remaining to-do      : all other specs
"""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

ID_RE = re.compile(r"\[id:(\d+)\]")


def get_specs() -> list[dict]:
    tmp = Path("/tmp/ginkgo_dryrun.json")
    try:
        cmd = ["ginkgo", "run", "--dry-run", f"--json-report={tmp}", "./tests/e2e/"]
        res = subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
        if res.returncode != 0:
            sys.exit(f"ginkgo --dry-run failed:\n{res.stderr}")
        with open(tmp) as f:
            data = json.load(f)
        return [
            s
            for suite in data
            for s in suite.get("SpecReports", [])
            if s.get("LeafNodeType") == "It"
        ]
    finally:
        if tmp.exists():
            tmp.unlink()


def main():
    parser = argparse.ArgumentParser(description="HyperFleet E2E Parity Scoreboard")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    parser.add_argument("--details", action="store_true", help="Show test details")
    args = parser.parse_args()

    buckets = {"validated": [], "sanity": [], "deferred": [], "na": [], "todo": []}

    for s in get_specs():
        labels = set(s.get("LeafNodeLabels") or [])
        for c in s.get("ContainerHierarchyLabels") or []:
            labels.update(c)

        m = ID_RE.search(s.get("LeafNodeText", ""))
        item = {
            "file": Path(s.get("LeafNodeLocation", {}).get("FileName", "")).name,
            "title": s.get("LeafNodeText", ""),
            "id": m.group(1) if m else "",
        }

        if "hyperfleet-validated" in labels:
            buckets["validated"].append(item)
        elif "hyperfleet-sanity" in labels:
            buckets["sanity"].append(item)
        elif "hyperfleet-deferred" in labels:
            buckets["deferred"].append(item)
        elif "hyperfleet-na" in labels:
            buckets["na"].append(item)
        else:
            buckets["todo"].append(item)

    total = sum(len(v) for v in buckets.values())
    viable = total - len(buckets["na"])
    pct = round((len(buckets["validated"]) / viable * 100), 1) if viable > 0 else 0.0

    summary = {
        "total_specs": total,
        "viable_specs": viable,
        "validated": len(buckets["validated"]),
        "sanity": len(buckets["sanity"]),
        "deferred": len(buckets["deferred"]),
        "not_applicable": len(buckets["na"]),
        "remaining_todo": len(buckets["todo"]),
        "parity_percentage": pct,
    }

    if args.json:
        if args.details:
            summary["details"] = buckets
        print(json.dumps(summary, indent=2))
        return

    print("\n=======================================================")
    print("      ROSA HyperFleet v1 E2E Parity Scoreboard         ")
    print("=======================================================")
    print(f" Total E2E Ginkgo Specs:       {summary['total_specs']}")
    print(f" Not Applicable (in code):     {summary['not_applicable']}")
    print(f" -----------------------------------------------------")
    print(f" Viable HyperFleet Specs:      {summary['viable_specs']}")
    print(f"   • Validated (passing):      {summary['validated']:3d}  [hyperfleet-validated]")
    print(f"   • Sanity (standalone CRUD): {summary['sanity']:3d}  [hyperfleet-sanity]")
    print(f"   • Deferred (known gap):     {summary['deferred']:3d}  [hyperfleet-deferred]")
    print(f"   • Remaining To-Do:          {summary['remaining_todo']:3d}  (pending / unlabeled)")
    print(f" -----------------------------------------------------")
    print(f" Parity Progress:              {summary['parity_percentage']}% ({summary['validated']}/{summary['viable_specs']} validated)")
    print("=======================================================\n")

    if args.details:
        for cat in ["validated", "sanity", "deferred", "na"]:
            print(f"--- {cat.upper()} ({len(buckets[cat])}) ---")
            for item in buckets[cat]:
                tid = f"[id:{item['id']}] " if item["id"] else ""
                print(f"  • {item['file']}: {tid}{item['title']}")
            print()


if __name__ == "__main__":
    main()
