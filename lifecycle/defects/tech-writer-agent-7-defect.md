---
title: lifecycle/docs directory not auto-created when writing doc artifacts
type: defect
status: in-development
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: backend-developer
      who: agent
---

# lifecycle/docs directory not auto-created when writing doc artifacts

## Reproduction Steps

1. Start a project that includes a `docs` stage (dir: `lifecycle/docs`) in its config but whose project root does not yet contain a `lifecycle/docs/` directory.
2. Send `POST /api/p/testproject/artifacts` with `stage: "docs"`, `slug: "install-guide"`, and valid doc frontmatter.
3. Observe the response.

## Expected Behaviour

The server creates the `lifecycle/docs/` directory if it does not exist and writes the artifact file, returning HTTP 201 with the artifact path.

## Actual Behaviour

The server returns HTTP 400:

```
{"error":{"code":"invalid_path","message":"resolving parent directory: lstat /tmp/.../lifecycle/docs: no such file or directory"}}
```

The directory is never created and the artifact file is never written.

## Logs / Output

```
api_doc_create_test.go:40: expected status 201, got 400: {"error":{"code":"invalid_path","message":"resolving parent directory: lstat /private/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestDocCreate_OriginatingDoc2670955093/001/lifecycle/docs: no such file or directory"}}
--- FAIL: TestDocCreate_OriginatingDoc (0.18s)
--- FAIL: TestDocCreate_SourceLinkedDoc (0.19s)
--- FAIL: TestDocCreate_IndexerPicksUp (0.15s)
--- FAIL: TestDocCreate_GitCommit (0.14s)
```

The same error occurs for all four create tests and for the E2E Flow 07 TC3 (standalone doc creation via the UI), where `page.waitForResponse` for `POST /artifacts` times out after 20 s.

**Failing tests:** `TestDocCreate_OriginatingDoc`, `TestDocCreate_SourceLinkedDoc`, `TestDocCreate_IndexerPicksUp`, `TestDocCreate_GitCommit` (`tests/integration/api_doc_create_test.go`); `Flow 07 TC3` (`tests/e2e/flows/07-doc-new.spec.ts`).
