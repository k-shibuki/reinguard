// Issue label safety net: optional auto-label from Task form "### Type" + soft validation.
// Label lists are read from LABELS_POLICY_JSON (yq output of .reinguard/labels.yaml in CI).

const fs = require("fs");

const run = async () => {
  const issue = context.payload.issue;
  if (!issue || !issue.number) {
    core.setFailed(
      "issue-label-sync: missing issue in payload (expected issues event).",
    );
    return;
  }

  const labelsPath = process.env.LABELS_POLICY_JSON || "/tmp/labels.json";
  let labelsDoc;
  try {
    labelsDoc = JSON.parse(fs.readFileSync(labelsPath, "utf8"));
  } catch (e) {
    core.setFailed(
      `issue-label-sync: cannot read label config at ${labelsPath}: ${e.message}`,
    );
    return;
  }

  if (
    !labelsDoc.categories?.type?.labels?.length ||
    !labelsDoc.categories?.exception?.labels?.length
  ) {
    core.setFailed(
      `issue-label-sync: label config at ${labelsPath} is missing required categories (type/exception).`,
    );
    return;
  }

  const TYPE_LABELS = labelsDoc.categories.type.labels.map((x) => x.name);
  const EXCEPTION_LABELS = labelsDoc.categories.exception.labels.map(
    (x) => x.name,
  );
  const SCOPE_LABEL_NAMES = (
    labelsDoc.categories.scope?.labels || []
  ).map((x) => x.name);

  const { data: full } = await github.rest.issues.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issue.number,
  });
  const body = full.body || "";
  let labels = (full.labels || []).map((l) => l.name);

  const messages = [];

  /** Post informational comment once per identical body (avoids spam on repeated issues events). */
  const postInformationalComment = async (msgs) => {
    const commentBody =
      "## Issue label check (informational)\n\n" +
      msgs.map((m) => `- ${m}`).join("\n");

    const { data: comments } = await github.rest.issues.listComments({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issue.number,
      per_page: 100,
    });

    const alreadyPosted = comments.some(
      (c) => c.user?.type === "Bot" && c.body === commentBody,
    );
    if (alreadyPosted) {
      return;
    }

    await github.rest.issues.createComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issue.number,
      body: commentBody,
    });
  };

  const isEpicIssue = SCOPE_LABEL_NAMES.some((n) => labels.includes(n));

  // --- Epic (scope labels from SSOT): type labels should not be combined ---
  if (isEpicIssue) {
    const typeOnEpic = TYPE_LABELS.filter((t) => labels.includes(t));
    if (typeOnEpic.length > 0) {
      messages.push(
        `This Issue has a scope label (${SCOPE_LABEL_NAMES.filter((n) => labels.includes(n)).join(", ")}) but also type label(s): ${typeOnEpic.join(", ")}. For epic planning Issues, remove type labels.`,
      );
    }
    if (messages.length > 0) {
      await postInformationalComment(messages);
    }
    return;
  }

  const exceptionOnIssue = EXCEPTION_LABELS.filter((e) => labels.includes(e));
  if (exceptionOnIssue.length > 0) {
    messages.push(
      `PR-only exception label(s) on an Issue: ${exceptionOnIssue.join(", ")}. Remove them (exception labels apply to PRs, not Issues).`,
    );
  }

  // --- Task form: ### Type → optional auto-label (skip if PR-only exception labels present) ---
  const typeMatch = body.match(/^### Type\s*\n+\s*(\S+)/m);
  const dropdownType = typeMatch ? typeMatch[1].trim() : null;

  if (!dropdownType && /###\s*Type/i.test(body)) {
    core.info(
      "issue-label-sync: found '### Type' in body but could not extract value (markdown layout may have changed).",
    );
  }

  if (
    exceptionOnIssue.length === 0 &&
    dropdownType &&
    TYPE_LABELS.includes(dropdownType)
  ) {
    if (!labels.includes(dropdownType)) {
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issue.number,
        labels: [dropdownType],
      });
      labels = [...labels, dropdownType];
    }
  }

  const typeHits = TYPE_LABELS.filter((t) => labels.includes(t));
  if (typeHits.length === 0) {
    messages.push(
      `No **type** label found. Add one of: ${TYPE_LABELS.join(", ")} (or use the **Task** Issue Form so the Type field can be applied).`,
    );
  } else if (typeHits.length > 1) {
    messages.push(
      `Multiple type labels: ${typeHits.join(", ")}. Keep exactly one.`,
    );
  }

  if (
    dropdownType &&
    TYPE_LABELS.includes(dropdownType) &&
    typeHits.length === 1 &&
    dropdownType !== typeHits[0]
  ) {
    messages.push(
      `Type field (\`${dropdownType}\`) does not match the type label (\`${typeHits[0]}\`). Align the GitHub label with the form selection.`,
    );
  }

  if (messages.length > 0) {
    await postInformationalComment(messages);
  }
};

return await run();
