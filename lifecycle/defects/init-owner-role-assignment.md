---
title: Init Does Not Assign Owner Roles to First User
type: defect
status: in-development
lineage: init-owner-role-assignment
created: "2026-05-12T10:34:05+10:00"
priority: normal
labels:
    - defect
    - onboarding
    - backend
release: KC-Release1
---

# Init Does Not Assign Owner Roles to First User

## Reproduction Steps

1. Run `kaos-control init` to perform first-time setup.
2. Enter user credentials when prompted to create the initial admin/owner account.
3. Inspect the auth configuration or user record for the created user.

## Expected Behaviour

The user created during `kaos-control init` should automatically be designated as the project owner and assigned the following roles: `product-owner`, `analyst`, `reviewer`, `approver`, `devops`.

## Actual Behaviour

The initial user is created without owner designation and without any pre-assigned roles. Roles must be manually configured after setup, which is error-prone and creates a broken initial state where no one has the necessary permissions to begin using the tool.
