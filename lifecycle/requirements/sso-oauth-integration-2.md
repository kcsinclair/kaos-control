---
title: SSO via OAuth 2.0 / OpenID Connect
type: requirement
status: blocked
lineage: sso-oauth-integration
created: "2026-05-11"
priority: normal
parent: lifecycle/ideas/sso-oauth-integration.md
labels:
    - feature
    - security
    - integration
    - backend
assignees:
    - role: product-owner
      who: agent
---

# SSO via OAuth 2.0 / OpenID Connect

## Problem

kaos-control currently supports only local username/password authentication (argon2id hashes stored in SQLite). Organisations that already use an identity provider (Google Workspace, GitHub, Microsoft Entra ID) must create and manage a separate set of credentials for kaos-control, which:

1. Increases the credential surface area for users and admins.
2. Prevents enforcement of the organisation's existing MFA and conditional-access policies.
3. Adds friction to onboarding — every new team member needs a manually-created local account.

## Goals / Non-goals

### Goals

- G1: Allow users to authenticate via any OAuth 2.0 / OpenID Connect provider configured by the admin.
- G2: Support at least three concrete providers out of the box: Google, GitHub, and Microsoft Entra ID.
- G3: Automatically link an external identity to an existing local user account when the verified email matches.
- G4: Allow new user accounts to be provisioned on first SSO login when the admin enables auto-provisioning.
- G5: Preserve the existing local-password login flow as a parallel option — SSO is additive, not a replacement.
- G6: Expose provider-specific login buttons on the frontend login page with correct redirect-based auth flow.
- G7: Store provider configuration (client ID, client secret, scopes, endpoints) in `config.yaml` — no external database required.

### Non-goals

- NG1: SAML 2.0 support (may be addressed in a future artifact).
- NG2: Multi-tenancy or per-project provider scoping — providers are configured at the application level.
- NG3: Replacing or deprecating local-password auth.
- NG4: Group/role synchronisation from the IdP (roles remain manually assigned in `config.yaml`).
- NG5: Custom or self-hosted OIDC provider support beyond the three named providers (generic OIDC may work but is not a test target).

## Detailed Requirements

### Functional

#### FR-1: Provider configuration

The `auth` section of the app-level `config.yaml` must accept an `oauth_providers` list. Each entry must include:

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Unique provider identifier (e.g. `google`, `github`, `entra`) |
| `display_name` | string | no | Label shown on the login button; defaults to `name` |
| `client_id` | string | yes | OAuth client ID |
| `client_secret` | string | yes | OAuth client secret |
| `issuer` | string | yes for OIDC | OIDC issuer URL (used for auto-discovery via `.well-known/openid-configuration`) |
| `auth_url` | string | yes if no `issuer` | Authorization endpoint (GitHub, which is not OIDC) |
| `token_url` | string | yes if no `issuer` | Token endpoint |
| `userinfo_url` | string | yes if no `issuer` | Userinfo endpoint |
| `scopes` | []string | no | Defaults to `["openid", "email", "profile"]` |
| `auto_provision` | bool | no | Create local user on first login if `true`; default `false` |

The server must validate provider config at startup and fail fast on missing required fields.

#### FR-2: OAuth callback handler

- A new route `GET /api/auth/oauth/{provider}/callback` must handle the OAuth redirect.
- The handler must validate the `state` parameter against a server-side CSRF token stored in a short-lived (10-minute) entry to prevent CSRF and replay attacks.
- On success, exchange the authorization code for tokens, extract the user's verified email from the ID token (OIDC) or userinfo endpoint (GitHub), and either match to an existing local user or auto-provision one (if enabled).
- On success, create a session and set cookies identically to `handleLogin`.

#### FR-3: OAuth initiation endpoint

- `GET /api/auth/oauth/{provider}/login` must redirect the user to the provider's authorization URL with correct `client_id`, `redirect_uri`, `scope`, `state`, and `response_type=code`.
- The `redirect_uri` must be deterministic from the server's external URL (configurable or inferred from the `Host` header) plus the callback path.

#### FR-4: Account linking

- When a user authenticates via SSO and the provider returns a verified email that matches an existing `users.email`, the session is created for that existing account — no duplicate account is created.
- When `auto_provision` is `false` and no matching local user exists, return an error page/message ("Account not found — contact your administrator").
- When `auto_provision` is `true`, create a new local user with the provider's email and display name, no password hash (password-less account), and create a session.

#### FR-5: Password-less accounts

- The `users` table must allow a `NULL` or empty `password_hash` to represent accounts that authenticate exclusively via SSO.
- Local login (`POST /api/auth/login`) must reject password-less accounts with a clear error ("This account uses SSO — use the provider login button").

#### FR-6: Frontend login page

- The login page must display one button per configured OAuth provider, labelled with `display_name`.
- Clicking a provider button must navigate to `/api/auth/oauth/{provider}/login`.
- After successful OAuth callback, the server must redirect to the SPA's post-login route (the user's original destination or `/`).
- Error states (provider unreachable, account not found, email not verified) must be surfaced to the user via a query-parameter-based error display on the login page.

#### FR-7: Logout

- Logout (`POST /api/auth/logout`) must continue to work identically for SSO-authenticated sessions — it destroys the local session. Provider-side logout (front-channel or back-channel) is out of scope.

### Non-functional

#### NFR-1: Security

- All OAuth state tokens must be cryptographically random (≥32 bytes) and single-use.
- The callback handler must enforce exact `redirect_uri` matching.
- Provider client secrets must not be logged or exposed via any API endpoint.
- HTTPS must be enforced when OAuth providers are configured (providers will reject plain-HTTP redirect URIs in production).

#### NFR-2: Startup validation

- If any configured provider has invalid or missing fields, the server must refuse to start and log a clear error identifying the misconfigured provider.

#### NFR-3: No new runtime dependencies

- The implementation should use Go's `net/http` client and the `encoding/json` stdlib for token exchange and userinfo fetching. No third-party OAuth library is required — the flow is standard enough to implement directly.

#### NFR-4: Database migration

- The `users` table schema change (nullable `password_hash`) must be applied via an idempotent migration in `auth.Store.createSchema()`, consistent with the existing migration pattern.
- A new `oauth_states` table must store pending state tokens with expiry for server-side validation.

## Acceptance Criteria

- [ ] At least one OIDC provider (Google or Entra) and GitHub can be configured in `config.yaml` and the server starts without error.
- [ ] Clicking a provider button on the login page redirects to the correct provider authorization URL.
- [ ] After authorizing with the provider, the user is redirected back and a valid session is created.
- [ ] An existing local user whose email matches the provider email is logged in without creating a duplicate account.
- [ ] With `auto_provision: true`, a new user is created on first SSO login and a session is established.
- [ ] With `auto_provision: false` and no matching local user, a clear error is shown.
- [ ] Password-less (SSO-only) accounts cannot log in via the local email/password form.
- [ ] Local-password login continues to work for users who have a password set.
- [ ] OAuth state tokens are validated and single-use; replaying a callback URL fails.
- [ ] Provider client secrets do not appear in any API response or log output.
- [ ] The server refuses to start if a provider entry is missing required fields.
- [ ] `POST /api/auth/logout` destroys SSO-initiated sessions correctly.
- [ ] [[sso-oauth-integration]] idea is fully addressed.

## Open Questions

1. **External URL configuration**: Should the server's external URL (used to build `redirect_uri`) be an explicit config field (e.g. `server.external_url`), or inferred from the `Host` header on each request? An explicit field is safer for reverse-proxy deployments.
2. **Provider-side token storage**: Should we store the provider's access/refresh tokens in the database for future API calls (e.g. fetching user avatar), or discard them after extracting the email? Storing them adds complexity and a security surface; discarding them keeps the implementation minimal.
3. **Admin UI for provider management**: Should provider configuration remain config-file-only, or should there be an admin UI to add/remove providers at runtime? Config-file-only is simpler and consistent with the current auth config pattern.
