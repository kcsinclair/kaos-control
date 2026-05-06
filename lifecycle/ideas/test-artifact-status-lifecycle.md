---
title: Test Artifact Status Lifecycle
type: idea
status: draft
lineage: test-artifact-status-lifecycle
created: "2026-04-28T10:02:27+10:00"
priority: low
labels:
    - test
    - testing
    - qa
    - workflow
    - artefacts
---

# Test Artifact Status Lifecycle

## this might be redundant now.... need to check.

Test artifacts need a well-defined status lifecycle that mirrors the broader workflow model. A test begins in `draft` status while being authored, then transitions to `approved` once reviewed. Only approved tests are eligible to be executed by the QA agent.

When the QA agent runs an approved test, it may raise defects as a result. Once defect-raising is complete, the agent runner should automatically transition the test artifact back to `approved` status, leaving it ready for the next run. The raised defects stand as the actionable next steps for developers.

This cycle — approved → running → approved (with defects as side-effects) — ensures tests are never left in a terminal or ambiguous state after a QA run, and that the defect artifacts serve as the clear queue of work to address before the next test execution.
