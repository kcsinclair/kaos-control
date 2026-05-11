---
title: SSO with OAuth Integrations
type: idea
status: approved
lineage: sso-oauth-integration
created: "2026-05-11T12:21:53+10:00"
priority: normal
labels:
    - feature
    - security
    - integration
    - backend
---

# SSO with OAuth Integrations

The system should support Single Sign-On (SSO) via OAuth 2.0 / OpenID Connect integrations, allowing users to authenticate using external identity providers (e.g. Google, GitHub, Microsoft Entra ID) rather than — or in addition to — the existing username/password flow.

This would involve adding an OAuth callback handler to the Go HTTP layer, storing provider tokens and mapping external identities to local user accounts, and updating the session management to handle provider-issued tokens alongside the existing argon2id-based auth.

The frontend login flow would need to surface provider-specific login buttons and handle redirect-based auth flows, with appropriate error states and post-login redirects back to the user's intended destination.
