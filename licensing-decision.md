# Licensing decision — kaos-control

This document records the open-source licence chosen for kaos-control and the
reasoning behind it. It is intended for future readers (contributors, maintainers,
auditors) who want to understand *why* the project is AGPLv3 rather than something
more permissive.

## Decision

**AGPLv3** (GNU Affero General Public License, version 3) — full text in
[LICENSE](LICENSE).

Outside contributions are accepted under a **DCO sign-off** model — see
[CONTRIBUTING.md](CONTRIBUTING.md).

A separate commercial licence may be made available to organisations that
cannot accept the AGPL terms; the project is single-author copyright, which
keeps that option open.

## The question being answered

> *"What open source licence is best for this project? I believe in open source
> but do not want a large commercial organisation to take it and start selling
> it."*

The classic permissive licences (MIT, Apache 2, BSD) explicitly allow what is
to be avoided: anyone can take the code, modify it, host it as a paid service,
and never give back. Even GPLv3 has a well-known **SaaS loophole** — a cloud
provider can run modified code as a service forever without ever distributing
a binary, so the copyleft never triggers (this is the pattern that drove
MongoDB, Elastic, Redis and HashiCorp away from OSI-approved licences).

Three licences are actually built for the "don't let cloud providers commodify
this" concern:

| Licence | OSI-approved open source? | Stops a cloud provider rehosting it? | Cost |
|---|---|---|---|
| **AGPLv3** | ✅ yes | ✅ yes — network use counts as distribution; they must publish their fork | Some enterprises ban AGPL outright in procurement |
| **BUSL 1.1** (Business Source Licence) | ❌ source-available, not OSS | ✅ yes — production-as-a-service explicitly disallowed for N years | Loses the "open source" badge until the change date |
| **PolyForm Noncommercial / Shield** | ❌ source-available | ✅ yes — bans commercial use entirely (Noncommercial) or competing services (Shield) | Even more restrictive than BUSL; smaller mind-share |

## Why AGPLv3 was chosen

1. **The project's stated value is open source.** AGPL is real open source —
   OSI-approved, FSF-blessed. BUSL and PolyForm are source-available; calling
   them "open source" invites avoidable disputes.

2. **The threat model fits.** The concern is *"someone takes this and sells
   it as a hosted service."* AGPL's network-use clause is the specific lever
   invented for that case. Anyone who rehosts kaos-control as a SaaS must
   publish their full source, including their modifications. That removes
   the bulk of the commercial-takeover incentive.

3. **kaos-control is end-user software, not a library.** AGPL's
   "scary-for-enterprise-consumers" reputation is mostly about *libraries*
   linked into proprietary code. kaos-control is a standalone server you
   run; the AGPL boundary is at the network edge, which is exactly where
   you want it.

4. **Commercial optionality is preserved.** As the sole copyright holder,
   the maintainer can also offer a commercial licence to anyone who cannot
   accept AGPL terms — the standard "open core" / dual-licence pattern
   (Sentry pre-BUSL, GitLab, Qt). The DCO sign-off requirement on
   contributions keeps the relicensing right intact for original code, and
   contributor copyright stays with the contributor (no CLA assignment).

5. **It can be relicensed later.** A sole copyright holder can move from
   AGPL to BUSL or to a commercial-only model later if the project's
   needs change. The reverse (starting permissive and tightening later)
   is the move that burned trust at Elastic, HashiCorp and Redis.

## When BUSL would have been the right call instead

- If the maintainer could clearly see fundraising / building a company
  around this within 1–2 years and needed explicit "you can't compete with
  our hosted service" language for VC due diligence.
- If "monetisable optionality" mattered more than the OSI-approved label.

Neither condition holds at the time of decision (2026-05-06).

## Practical setup

The AGPL decision is implemented across the following files:

- [LICENSE](LICENSE) — verbatim AGPLv3 text from <https://www.gnu.org/licenses/agpl-3.0.txt>.
- [README.md](README.md) — Licence section updated to AGPLv3 with a contact
  pointer for commercial licences.
- [CONTRIBUTING.md](CONTRIBUTING.md) — DCO sign-off requirement
  (`git commit -s`); developercertificate.org reference; outside contribution
  workflow.

## Future re-evaluation

This decision should be revisited if any of the following becomes true:

- Adoption is materially throttled by enterprise AGPL bans.
- A commercial entity is being formed around the project.
- A cloud provider attempts to rehost kaos-control and the AGPL clause
  needs to be tested in practice.

Decision date: 2026-05-06.
