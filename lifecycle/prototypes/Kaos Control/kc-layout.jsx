// ── kc-layout.jsx — AppHeader, AppSidebar, RightPane ──

function AppHeader({ project, wsStatus, theme, onToggleTheme, onNavigate, activeAgent }) {
  return (
    <header style={{
      height: 48, display: 'flex', alignItems: 'center', gap: 0,
      background: 'var(--header-bg)', borderBottom: '1px solid var(--border-subtle)',
      padding: '0 16px', flexShrink: 0, zIndex: 100,
    }}>
      {/* Brand */}
      <button onClick={() => onNavigate('projects')} style={{
        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
        display: 'flex', alignItems: 'center', gap: 8, marginRight: 20,
      }}>
        <span style={{
          width: 28, height: 28, borderRadius: 'var(--r-md)',
          background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center',
          boxShadow: '0 0 12px var(--accent-glow)',
        }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
            <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/>
            <line x1="12" y1="22" x2="12" y2="15.5"/>
            <polyline points="22 8.5 12 15.5 2 8.5"/>
          </svg>
        </span>
        <span style={{ fontWeight: 700, fontSize: 'var(--text-md)', letterSpacing: '-0.01em', color: 'var(--text)' }}>
          kaos<span style={{ color: 'var(--accent)' }}>-</span>control
        </span>
      </button>

      {/* Project breadcrumb */}
      {project && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: 'var(--text-muted)', fontSize: 'var(--text-sm)' }}>
          <span style={{ color: 'var(--border)' }}>/</span>
          <span style={{ color: 'var(--text)', fontFamily: 'var(--font-mono)', fontWeight: 500 }}>{project}</span>
        </div>
      )}

      <div style={{ flex: 1 }} />

      {/* Active agent badge */}
      {activeAgent && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 6, marginRight: 12,
          padding: '4px 10px', borderRadius: 'var(--r-full)',
          background: 'var(--node-plan-backend)', color: 'oklch(0.08 0 0)',
          fontSize: 'var(--text-xs)', fontWeight: 600, fontFamily: 'var(--font-mono)',
          animation: 'kc-pulse 2s ease infinite',
        }}>
          <Spinner size={10} />
          {activeAgent.agent} running
        </div>
      )}

      {/* WS status */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginRight: 16,
        fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
        <WsStatusDot status={wsStatus} />
        <span>{wsStatus}</span>
      </div>

      {/* Theme toggle */}
      <button className="btn btn-ghost btn-icon btn-sm" onClick={onToggleTheme}
        title="Toggle theme" style={{ color: 'var(--text-muted)' }}>
        <Icon name={theme === 'studio' ? 'moon' : 'sun'} size={14} />
      </button>
    </header>
  );
}

const NAV_ITEMS = [
  { id: 'graph',     icon: 'graph',    label: 'Graph',     shortcut: 'G' },
  { id: 'graph2d',   icon: 'filter',   label: '2D Graph',  shortcut: '' },
  { id: 'artifacts', icon: 'list',     label: 'Artifacts', shortcut: 'A' },
  { id: 'agents',    icon: 'bot',      label: 'Agents',    shortcut: '' },
  { id: 'config',    icon: 'settings', label: 'Config',    shortcut: '' },
];

const STAGES = [
  { id: 'all',             label: 'All stages',      count: 11 },
  { id: 'ideas',           label: 'Ideas',           count: 4 },
  { id: 'requirements',    label: 'Requirements',    count: 2 },
  { id: 'backend-plans',   label: 'Backend plans',   count: 1 },
  { id: 'frontend-plans',  label: 'Frontend plans',  count: 1 },
  { id: 'dev-plans',       label: 'Dev plans',       count: 1 },
  { id: 'releases',        label: 'Releases',        count: 1 },
  { id: 'sprints',         label: 'Sprints',         count: 1 },
];

function AppSidebar({ activeTab, onTabChange, stageFilter, onStageFilter, agentCount, compact }) {
  const [savedViewsOpen, setSavedViewsOpen] = React.useState(true);

  return (
    <aside style={{
      width: compact ? 52 : 220, flexShrink: 0,
      background: 'var(--sidebar-bg)', borderRight: '1px solid var(--border-subtle)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
      transition: 'width var(--dur-base) var(--ease)',
    }}>
      {/* Nav items */}
      <nav style={{ padding: '8px 6px', display: 'flex', flexDirection: 'column', gap: 2 }}>
        {NAV_ITEMS.map(item => {
          const active = activeTab === item.id;
          return (
            <button key={item.id} onClick={() => onTabChange(item.id)}
              title={compact ? item.label : undefined}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: compact ? '8px 12px' : '7px 10px', borderRadius: 'var(--r-md)',
                background: active ? 'color-mix(in oklch, var(--accent) 15%, transparent)' : 'transparent',
                border: 'none', cursor: 'pointer', width: '100%', textAlign: 'left',
                color: active ? 'var(--accent)' : 'var(--text-muted)',
                fontSize: 'var(--text-sm)', fontWeight: active ? 600 : 400,
                transition: 'all var(--dur-fast) var(--ease)',
                justifyContent: compact ? 'center' : 'flex-start',
              }}>
              <Icon name={item.icon} size={15} />
              {!compact && <span style={{ flex: 1 }}>{item.label}</span>}
              {!compact && item.id === 'agents' && agentCount > 0 && (
                <span style={{
                  background: 'var(--accent)', color: 'var(--accent-text)',
                  borderRadius: 'var(--r-full)', padding: '0 6px', fontSize: 10, fontWeight: 700,
                }}>{agentCount}</span>
              )}
            </button>
          );
        })}
      </nav>

      {!compact && (
        <>
          <div style={{ height: 1, background: 'var(--border-subtle)', margin: '4px 12px' }} />

          {/* Stage filters */}
          <div style={{ padding: '8px 6px', flex: 1, overflow: 'auto' }}>
            <div style={{
              fontSize: 10, fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase',
              color: 'var(--text-faint)', padding: '4px 6px 6px', fontFamily: 'var(--font-mono)',
            }}>Stages</div>
            {STAGES.map(s => (
              <button key={s.id} onClick={() => onStageFilter(s.id)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '5px 8px', borderRadius: 'var(--r-md)',
                  background: stageFilter === s.id ? 'var(--bg-elevated)' : 'transparent',
                  border: 'none', cursor: 'pointer', width: '100%', textAlign: 'left',
                  color: stageFilter === s.id ? 'var(--text)' : 'var(--text-muted)',
                  fontSize: 'var(--text-sm)', transition: 'all var(--dur-fast) var(--ease)',
                }}>
                <span style={{ flex: 1 }}>{s.label}</span>
                <span style={{
                  fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                  background: 'var(--bg-elevated)', padding: '1px 5px', borderRadius: 'var(--r-full)',
                }}>{s.count}</span>
              </button>
            ))}
          </div>

          <div style={{ height: 1, background: 'var(--border-subtle)', margin: '4px 12px' }} />

          {/* Saved views */}
          <div style={{ padding: '8px 6px' }}>
            <button onClick={() => setSavedViewsOpen(o => !o)} style={{
              display: 'flex', alignItems: 'center', gap: 6, width: '100%',
              background: 'none', border: 'none', cursor: 'pointer',
              fontSize: 10, fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase',
              color: 'var(--text-faint)', padding: '4px 6px 6px', fontFamily: 'var(--font-mono)',
            }}>
              <Icon name="chevron" size={10} style={{ transform: savedViewsOpen ? 'rotate(90deg)' : 'rotate(0)' }} />
              Saved Views
            </button>
            {savedViewsOpen && ['Auth flow', 'Release r-2026.05', 'In-development'].map(v => (
              <div key={v} style={{
                padding: '5px 8px', fontSize: 'var(--text-sm)', color: 'var(--text-muted)',
                borderRadius: 'var(--r-md)', cursor: 'pointer', transition: 'all var(--dur-fast) var(--ease)',
              }}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-elevated)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
              >{v}</div>
            ))}
          </div>
        </>
      )}
    </aside>
  );
}

function RightPane({ artifact, onClose, onEdit, onRunAgent }) {
  if (!artifact) {
    return (
      <aside style={{
        width: 280, flexShrink: 0, borderLeft: '1px solid var(--border-subtle)',
        background: 'var(--bg)', display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        <div style={{ color: 'var(--text-faint)', fontSize: 'var(--text-sm)', textAlign: 'center', padding: 24 }}>
          <Icon name="eye" size={24} style={{ opacity: 0.3 }} />
          <div style={{ marginTop: 8 }}>Click a node<br/>to inspect</div>
        </div>
      </aside>
    );
  }

  return (
    <aside style={{
      width: 280, flexShrink: 0, borderLeft: '1px solid var(--border-subtle)',
      background: 'var(--bg)', display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        padding: '12px 14px', borderBottom: '1px solid var(--border-subtle)',
        display: 'flex', alignItems: 'flex-start', gap: 8,
      }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontWeight: 600, fontSize: 'var(--text-md)', marginBottom: 6,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {artifact.title}
          </div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            <TypeBadge type={artifact.type} />
            <StatusBadge status={artifact.status} />
          </div>
        </div>
        <button className="btn btn-ghost btn-icon btn-sm" onClick={onClose}>
          <Icon name="close" size={13} />
        </button>
      </div>

      {/* Path */}
      <div style={{ padding: '8px 14px', borderBottom: '1px solid var(--border-subtle)' }}>
        <div style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', overflowWrap: 'break-word' }}>
          lifecycle/{artifact.path}
        </div>
      </div>

      {/* Quick actions */}
      <div style={{ padding: '10px 14px', borderBottom: '1px solid var(--border-subtle)', display: 'flex', gap: 6 }}>
        <button className="btn btn-ghost btn-sm" onClick={() => onEdit(artifact)} style={{ flex: 1, justifyContent: 'center' }}>
          <Icon name="edit" size={12} /> Edit
        </button>
        <button className="btn btn-ghost btn-sm" onClick={() => onRunAgent(artifact)} style={{ flex: 1, justifyContent: 'center' }}>
          <Icon name="bot" size={12} /> Agent
        </button>
      </div>

      {/* Frontmatter */}
      <div style={{ padding: '12px 14px', flex: 1, overflow: 'auto', fontSize: 'var(--text-sm)' }}>
        {[
          { label: 'Lineage', value: artifact.lineage },
          { label: 'Release', value: artifact.release },
          { label: 'Sprint', value: artifact.sprint },
          { label: 'Updated', value: artifact.updatedAt ? new Date(artifact.updatedAt).toLocaleDateString() : null },
        ].filter(f => f.value).map(f => (
          <div key={f.label} style={{ marginBottom: 10 }}>
            <div style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
              textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 3 }}>{f.label}</div>
            <div style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>{f.value}</div>
          </div>
        ))}

        {artifact.labels && artifact.labels.length > 0 && (
          <div style={{ marginBottom: 10 }}>
            <div style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
              textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 5 }}>Labels</div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
              {artifact.labels.map(l => <LabelChip key={l} label={l} />)}
            </div>
          </div>
        )}

        {artifact.assignees && artifact.assignees.length > 0 && (
          <div style={{ marginBottom: 10 }}>
            <div style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
              textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 6 }}>Assignees</div>
            {artifact.assignees.map((a, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                <Avatar email={a.who} size={20} />
                <div>
                  <div style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text)' }}>{a.who}</div>
                  <div style={{ fontSize: 10, color: 'var(--text-faint)' }}>{a.role}</div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Body preview */}
        {artifact.body && (
          <div style={{ marginTop: 12 }}>
            <div style={{ height: 1, background: 'var(--border-subtle)', marginBottom: 12 }} />
            <div style={{ fontSize: 'var(--text-xs)', color: 'var(--text-muted)', lineHeight: 1.7, whiteSpace: 'pre-wrap' }}>
              {artifact.body.slice(0, 300)}{artifact.body.length > 300 ? '…' : ''}
            </div>
          </div>
        )}
      </div>
    </aside>
  );
}

Object.assign(window, { AppHeader, AppSidebar, RightPane });
