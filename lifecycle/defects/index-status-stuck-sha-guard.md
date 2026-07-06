---
title: Approving an artefact doesn't stick — index status stuck vs disk (content-hash skip-guard locks stale status)
type: defect
status: done
lineage: index-status-stuck-sha-guard
created: "2026-07-06T00:00:00+10:00"
priority: high
labels:
    - defect
    - index
    - workflow
    - reliability
assignees:
    - role: backend-developer
      who: agent
---

# Approving an artefact doesn't stick — index status stuck vs disk

## Summary

A status change (e.g. draft → approved) is written to disk and committed to git,
but the SQLite index keeps serving the **old** status, so the app shows the
change reverting. Reproduced cleanly (survives a server restart) on
`lifecycle/tests/idea-archiving-5-test.md`.

## Reproduction Steps

1. Have an artefact whose index row's `status` has drifted from the file on disk
   while the stored `body_sha256` still equals the file's hash (see Root Cause —
   arises from a concurrent-write / external-sync race).
2. Change its status in the app (or edit the file); the transition writes the new
   status to disk and records a `status_transition` event.
3. Observe: disk + git show the new status, but the app (reading the index) still
   shows the old one. Restarting the server does **not** fix it.

Observed state on the affected artefact: disk + git = `approved`; index row =
`status: draft`, `body_sha256` = hash of the **approved** file, row untouched
since the morning's file mtime.

## Root Cause

`IndexFile` ([internal/index/index.go](../../internal/index/index.go)) has a
content-hash skip-guard: if the stored `body_sha256` equals the current file's
hash, it returns early without re-indexing (an M4 startup-scan optimisation to
avoid re-parsing unchanged files). This treats "content unchanged since last
index" as "index row is current" — but the two can diverge: if the row's
`status` column ever drifts from the file, the guard **locks the stale status in
permanently**, because the file's hash keeps matching on every subsequent index
(startup scan, watcher event, *and* the transition's own `IndexFile`). The
`status_transition` event still fires (it is recorded regardless of the skipped
Upsert), producing the illusion the change saved.

How the row first desyncs (status paired with the *other* status's content hash)
could not be reproduced deterministically from static analysis — `INSERT OR
REPLACE` writes `status` and `body_sha256` together from one parse, so a single
Upsert cannot produce it. It is most likely a concurrent-write race — the
affected file is also written by **external Obsidian-vault replication** (see
[[kaos-control-obsidian-index-desync]] context), i.e. writes the fsnotify watcher
sees in addition to kaos-control's own API writes. Left open for deeper
investigation; the fix below makes the amplifier self-healing regardless of
origin.

Secondary: `applyTransition`
([internal/http/transition.go](../../internal/http/transition.go)) discards the
`IndexFile` error (`_ = p.Idx.IndexFile(absPath)`), so a skipped/failed index
update is silent.

## Fix (implemented)

Make the skip-guard also require the stored `status` to match the freshly parsed
status before skipping. A pure content-hash match is not sufficient evidence the
row is current. This lets the next index (a transition, watcher event, or startup
scan) reconcile a drifted row — so **the user's next approve corrects it, and a
restart heals all stuck rows via the startup scan**. Performance is unaffected
(status is read in the same row query).

Regression test:
`internal/index/status_drift_heal_test.go` —
`TestIndexFile_HealsStatusDriftDespiteMatchingHash` seeds a row whose status has
drifted while the hash still matches, re-indexes the byte-unchanged file, and
asserts the status reconciles. Verified red without the fix, green with it.

## Follow-ups (not in this fix)

- Root-cause the *origin* of the status/hash desync (concurrent-write / external
  Obsidian-sync race).
- Stop discarding the `IndexFile` error in `applyTransition`.

## Deploy note

The fix ships in the Go binary — rebuild + restart to deploy; the startup scan
then self-heals the currently-stuck artefacts.
