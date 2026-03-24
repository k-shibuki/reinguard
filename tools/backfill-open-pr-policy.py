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


def gh_open_pulls_raw() -> str:
    """Fetch all open PRs; --paginate may emit multiple JSON arrays — parse below."""
    return subprocess.check_output(
        [
            "gh",
            "api",
            "repos/{owner}/{repo}/pulls",
            "-f",
            "state=open",
            "-f",
            "per_page=100",
            "--paginate",
        ],
        text=True,
    )


def parse_paginated_json_arrays(raw: str) -> list[dict]:
    """Concatenate top-level JSON arrays from gh api --paginate stdout."""
    decoder = json.JSONDecoder()
    idx = 0
    merged: list[dict] = []
    s = raw.strip()
    while idx < len(s):
        obj, end = decoder.raw_decode(s, idx)
        if isinstance(obj, list):
            merged.extend(item for item in obj if isinstance(item, dict))
        idx = end
        while idx < len(s) and s[idx].isspace():
            idx += 1
    return merged


def has_heading(body: str, title: str) -> bool:
    """Case-insensitive: PR body already has `## {title}` heading line."""
    pat = rf"(?im)^##\s+{re.escape(title)}\s*$"
    return bool(re.search(pat, body))


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

    if not has_heading(b, "Traceability"):
        close = extract_closes_line(b)
        if close:
            additions.append("## Traceability\n\n" + close + "\n")
        else:
            additions.append(
                "## Traceability\n\nCloses # (fill in — see Linked issues above if present)\n"
            )

    if not has_heading(b, "Risk / Impact"):
        additions.append(
            "## Risk / Impact\n\n"
            "- Affected area: see Summary.\n"
            "- Breaking change: no (update if yes).\n"
        )

    if not has_heading(b, "Rollback Plan"):
        additions.append("## Rollback Plan\n\nRevert this PR on main.\n")

    if not has_heading(b, "Definition of Done") and not has_heading(
        b, "Acceptance Criteria"
    ):
        additions.append(
            "## Definition of Done\n\n"
            "- [ ] (fill in verifiable criteria; see Issue if linked)\n"
        )

    if additions:
        sep = "\n\n" if b else ""
        b = b + sep + "\n\n".join(additions)
    return b


def main() -> int:
    raw = gh_open_pulls_raw()
    pulls = parse_paginated_json_arrays(raw)
    prs = [
        {
            "number": p["number"],
            "title": p["title"],
            "body": p.get("body") or "",
            "labels": [lb["name"] for lb in p.get("labels") or []],
        }
        for p in pulls
    ]
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
