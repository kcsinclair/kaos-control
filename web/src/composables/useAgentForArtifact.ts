// SPDX-License-Identifier: AGPL-3.0-or-later

// typeToAgent maps artefact type → the agent name responsible for processing it.
export const typeToAgent: Record<string, string> = {
  idea: 'requirements-analyst',
  ticket: 'planning-analyst',
  requirement: 'planning-analyst',
  'plan-backend': 'backend-developer',
  'plan-frontend': 'frontend-developer',
  'plan-test': 'test-developer',
  test: 'qa',
  doc: 'tech-writer',
}

// agentForArtifact returns the agent name to use for an artefact.
//
// For defect artefacts the agent is chosen by matching the first assignee role
// against the agents' configured roles — mirroring the AgentLaunchModal
// developer-defect branch.
//
// Returns null if no agent would handle this artefact (unrecognised type or
// defect with no matching assignee role).
export function agentForArtifact(
  artifact: { frontmatter: { type: string; assignees?: { role: string }[] } },
  agents: { name: string; roles: string[] }[],
): string | null {
  const { type, assignees } = artifact.frontmatter

  if (type === 'defect') {
    const assigneeRoles = new Set((assignees ?? []).map((a) => a.role))
    for (const agent of agents) {
      if (agent.roles.some((r) => assigneeRoles.has(r))) {
        return agent.name
      }
    }
    return null
  }

  return typeToAgent[type] ?? null
}
