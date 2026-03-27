// PR policy gate: Issue linkage, required sections, labels, base branch.
// Used by gate-policy in ci.yaml and by reusable pr-policy.yaml (workflow_call).
// Label lists are read from /tmp/labels.json (yq output of .reinguard/labels.yaml in CI).
// Exception contract: workflow-policy.mdc.

const fs = require("fs");

const run = async () => {
  const pr = context.payload.pull_request;
  if (!pr || !pr.number) {
    core.setFailed(
      "check-policy: missing pull_request in payload (expected pull_request event).",
    );
    return;
  }

  const labelsPath = process.env.LABELS_POLICY_JSON || "/tmp/labels.json";
  let labelsDoc;
  try {
    labelsDoc = JSON.parse(fs.readFileSync(labelsPath, "utf8"));
  } catch (e) {
    core.setFailed(
      `PR policy: cannot read label config at ${labelsPath} (run yq -o json .reinguard/labels.yaml > ${labelsPath}): ${e.message}`,
    );
    return;
  }

  if (
    !labelsDoc.categories?.type?.labels?.length ||
    !labelsDoc.categories?.exception?.labels?.length
  ) {
    core.setFailed(
      `PR policy: label config at ${labelsPath} is missing required categories (type/exception labels).`,
    );
    return;
  }

  const TYPE_LABELS = labelsDoc.categories.type.labels.map((x) => x.name);
  const EXCEPTION_LABELS = labelsDoc.categories.exception.labels.map(
    (x) => x.name,
  );

  // Always load body and labels via REST. Webhook / reusable-workflow payloads can
  // omit labels or serve a stale or truncated body compared to the live PR.
  const { data: full } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: pr.number,
  });
  const body = full.body || "";
  const labels = (full.labels || []).map((l) => l.name);
  const errors = [];
  const warnings = [];

  const isException = EXCEPTION_LABELS.some((l) => labels.includes(l));

  // --- 1. Issue linkage ---
  const hasIssueLink = /(?:closes|fixes|resolves)\s+#\d+/gi.test(body);

  if (!hasIssueLink) {
    if (isException) {
      const exceptionMatch = body.match(/## Exception[^]*?(?=\n## |$)/i);
      const exceptionBlock = exceptionMatch ? exceptionMatch[0] : "";

      const justMatch = exceptionBlock.match(/Justification:\s*(.+)/i);
      const justification = justMatch
        ? justMatch[1].replace(/<!--[^]*?-->/g, "").trim()
        : "";

      if (justification.length < 20) {
        errors.push(
          "**Issue linkage**: Exception label is set but justification is missing or too short. " +
            "Fill in the `## Exception` section with `- Justification: <reason>` (min 20 chars).",
        );
      } else {
        warnings.push(
          "**Issue linkage**: Exception — no Issue linked. Justification accepted.",
        );
      }
    } else {
      errors.push(
        "**Issue linkage**: PR body must contain `Closes #<number>` " +
          "(or `Fixes #N` / `Resolves #N`). " +
          "If this is an exception, add a label (`no-issue` / `hotfix`) " +
          "and fill in the `## Exception` section.",
      );
    }
  }

  // --- 2. Risk / Impact section ---
  const riskMatch = body.match(/## Risk \/ Impact[^]*?(?=\n## |$)/i);
  const riskContent = riskMatch
    ? riskMatch[0]
        .replace(/## Risk \/ Impact/i, "")
        .replace(/<!--[^]*?-->/g, "")
        .trim()
    : "";

  if (!riskMatch) {
    errors.push("**Risk / Impact**: Section is missing from the PR body.");
  } else if (riskContent.length < 5) {
    errors.push(
      "**Risk / Impact**: Section exists but appears empty. " +
        "Describe what could go wrong and who/what is affected.",
    );
  }

  // --- 3. Rollback Plan section ---
  const rollbackMatch = body.match(/## Rollback Plan[^]*?(?=\n## |$)/i);
  const rollbackContent = rollbackMatch
    ? rollbackMatch[0]
        .replace(/## Rollback Plan/i, "")
        .replace(/<!--[^]*?-->/g, "")
        .trim()
    : "";

  if (!rollbackMatch) {
    errors.push("**Rollback Plan**: Section is missing from the PR body.");
  } else if (rollbackContent.length < 3 || rollbackContent === "-") {
    errors.push(
      "**Rollback Plan**: Section exists but appears empty. " +
        'Describe how to revert if something goes wrong. For docs/test-only PRs, write "N/A".',
    );
  }

  // --- 4. Traceability section ---
  if (!/## Traceability/i.test(body)) {
    errors.push("**Traceability**: Section is missing from the PR body.");
  }

  // --- 5. PR title format (Conventional Commits) ---
  const titleRegex = new RegExp(
    `^(${TYPE_LABELS.join("|")})(\\(.+\\))?!?:\\s.+$`,
  );
  if (!titleRegex.test(full.title)) {
    errors.push(
      "**PR title**: Must follow Conventional Commits format: " +
        "`<type>(<scope>): <summary>`. " +
        "Valid types: " +
        TYPE_LABELS.join(", ") +
        ". " +
        "(`hotfix` is an exception label only, not a PR title type.)",
    );
  }

  // --- 6. Base branch check (HS-PR-BASE) ---
  if (full.base.ref !== "main" && full.base.ref !== "master") {
    errors.push(
      "**Base branch**: PR must target `main`. " +
        "Using `--base feat/<branch>` causes the PR to auto-close when the base branch is deleted on merge. " +
        "Document dependencies in the PR body instead.",
    );
  }

  // --- 7. Type label (exactly one) ---
  const typeHits = TYPE_LABELS.filter((t) => labels.includes(t));
  if (typeHits.length === 0) {
    errors.push(
      "**Type label**: PR must have exactly one type label " +
        "(" +
        TYPE_LABELS.map((t) => "`" + t + "`").join(", ") +
        "). " +
        'Add a label via `--label "<type>"` in `gh pr create`.',
    );
  } else if (typeHits.length > 1) {
    errors.push(
      "**Type label**: PR has multiple type labels (" +
        typeHits.join(", ") +
        "). " +
        "Keep exactly one; remove the rest.",
    );
  }

  // --- 8. Summary section (H1 or H2 for template compatibility) ---
  if (!/(?:^|\n)#{1,2}\s+Summary\b/im.test(body)) {
    errors.push("**Summary**: Section is missing from the PR body.");
  }

  // --- 9. Definition of Done section ---
  if (!/## (Acceptance Criteria|Definition of Done)/i.test(body)) {
    errors.push(
      "**Definition of Done**: Section is missing. " +
        'Add "## Definition of Done" with verifiable criteria.',
    );
  }

  // --- 10. Test plan section ---
  const testPlanMatch = body.match(/## Test plan[^]*?(?=\n## |$)/i);
  const testPlanContent = testPlanMatch
    ? testPlanMatch[0]
        .replace(/## Test plan/i, "")
        .replace(/<!--[^]*?-->/g, "")
        .trim()
    : "";

  if (!testPlanMatch) {
    errors.push("**Test plan**: Section is missing from the PR body.");
  } else if (testPlanContent.length < 5) {
    errors.push(
      "**Test plan**: Section exists but appears empty. " +
        "List concrete commands or checks and expected results.",
    );
  }

  // --- Report ---
  if (errors.length > 0) {
    const report = [
      "## PR Policy Check Failed",
      "",
      "### Errors (must fix)",
      ...errors.map((e) => `- ${e}`),
    ];
    if (warnings.length > 0) {
      report.push("", "### Warnings", ...warnings.map((w) => `- ${w}`));
    }
    report.push(
      "",
      "See [PR template](/.github/PULL_REQUEST_TEMPLATE.md) for required sections.",
    );
    core.setFailed(report.join("\n"));
  } else if (warnings.length > 0) {
    core.warning(warnings.join("\n"));
  }
};

return await run();
