#!/usr/bin/env python3
"""Backfill open PR bodies and labels for pr-policy.yaml (run from repo root).

Uses `gh api` (not `gh pr edit`) because some gh versions exit non-zero on `pr edit`
when Classic Projects GraphQL is deprecated.

Requires: gh CLI authenticated. Idempotent for sections already present.
"""
from __future__ import annotations

import json
import re
import subprocess
import sys


def gh_json(args: list[str]) -> object:
    out = subprocess.check_output(["gh"] + args, text=True)
    return json.loads(out)


def patch_pr_body(num: int, new_body: str) -> None:
    subprocess.run(
        [
            "gh",
            "api",
            "-X",
            "PATCH",
            f"repos/{{owner}}/{{repo}}/pulls/{num}",
            "--input",
            "-",
        ],
        input=json.dumps({"body": new_body}),
        text=True,
        check=True,
    )


def add_pr_label(num: int, label: str) -> None:
    subprocess.run(
        [
            "gh",
            "api",
            "-X",
            "POST",
            f"repos/{{owner}}/{{repo}}/issues/{num}/labels",
            "--input",
            "-",
        ],
        input=json.dumps({"labels": [label]}),
        text=True,
        check=True,
    )


def pr_type_from_title(title: str) -> str | None:
    m = re.match(
        r"^(feat|fix|refactor|perf|test|docs|build|ci|chore|style|revert)(\([^)]+\))?!?:",
        title,
    )
    return m.group(1) if m else None


def extract_closes_line(body: str) -> str | None:
    for line in body.splitlines():
        if re.search(r"(?i)(closes|fixes|resolves)\s+#\d+", line):
            return line.strip()
    return None


def ensure_sections(body: str) -> str:
    b = (body or "").replace("\r\n", "\n").rstrip()
    additions: list[str] = []

    if "## Traceability" not in b:
        close = extract_closes_line(b)
        if close:
            additions.append("## Traceability\n\n" + close + "\n")
        else:
            additions.append(
                "## Traceability\n\nCloses # (fill in — see Linked issues above if present)\n"
            )

    if "## Risk / Impact" not in b:
        additions.append(
            "## Risk / Impact\n\n"
            "- Affected area: see Summary.\n"
            "- Breaking change: no (update if yes).\n"
        )

    if "## Rollback Plan" not in b:
        additions.append("## Rollback Plan\n\nRevert this PR on main.\n")

    if additions:
        sep = "\n\n" if b else ""
        b = b + sep + "\n\n".join(additions)
    return b


def main() -> int:
    prs = gh_json(
        [
            "api",
            "repos/{owner}/{repo}/pulls?state=open&per_page=100",
            "--paginate",
            "--jq",
            "[.[] | {number: .number, title: .title, body: (.body // \"\"), labels: [.labels[].name]}]",
        ]
    )
    type_labels = {
        "feat",
        "fix",
        "refactor",
        "perf",
        "docs",
        "test",
        "ci",
        "build",
        "chore",
        "style",
        "revert",
    }

    for pr in prs:
        num = pr["number"]
        title = pr["title"]
        body = pr.get("body") or ""
        labels = set(pr.get("labels") or [])

        new_body = ensure_sections(body)
        if new_body != body:
            patch_pr_body(num, new_body)
            print(f"PR #{num}: updated body (policy sections)")

        want = pr_type_from_title(title)
        if want and want in type_labels and not labels & type_labels:
            add_pr_label(num, want)
            print(f"PR #{num}: added label {want}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
