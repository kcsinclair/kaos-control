// ── kc-views.jsx — Artifact list, Editor, Agents, Auth, Project picker ──

// ── Artifact List ──
function ArtifactListView({ artifacts, onSelect, onEdit }) {
  const [search, setSearch] = React.useState('');
  const [filterType, setFilterType] = React.useState('all');
  const [filterStatus, setFilterStatus] = React.useState('all');
  const [sortKey, setSortKey] = React.useState('updatedAt');
  const [sortDir, setSortDir] = React.useState('desc');

  const allTypes   = [...new Set(artifacts.map(a => a.type))];
  const allStatuses = [...new Set(artifacts.map(a => a.status))];

  const filtered = React.useMemo(() => {
    let list = artifacts.filter(a =>
      (filterType === 'all'   || a.type === filterType) &&
      (filterStatus === 'all' || a.status === filterStatus) &&
      (!search || a.title.toLowerCase().includes(search.toLowerCase()) ||
       a.path.toLowerCase().includes(search.toLowerCase()))
    );
    list = [...list].sort((a, b) => {
      let av = a[sortKey], bv = b[sortKey];
      if (!av) return 1; if (!bv) return -1;
      return sortDir === 'asc' ? (av > bv ? 1 : -1) : (av < bv ? 1 : -1);
    });
    return list;
  }, [artifacts, filterType, filterStatus, search, sortKey, sortDir]);

  function toggleSort(key) {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    else { setSortKey(key); setSortDir('asc'); }
  }

  const SortHeader = ({ label, k }) => (
    <th onClick={() => toggleSort(k)} style={{
      padding: '8px 12px', textAlign: 'left', cursor: 'pointer',
      color: sortKey === k ? 'var(--accent)' : 'var(--text-faint)',
      fontWeight: 600, fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)',
      textTransform: 'uppercase', letterSpacing: '0.06em', whiteSpace: 'nowrap',
      userSelect: 'none', borderBottom: '1px solid var(--border-subtle)',
    }}>
      {label} {sortKey === k ? (sortDir === 'asc' ? '↑' : '↓') : ''}
    </th>
  );

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px',
        borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-raised)', flexShrink: 0, flexWrap: 'wrap',
      }}>
        <div style={{ position: 'relative', flex: 1, minWidth: 200 }}>
          <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)',
            color: 'var(--text-faint)', pointerEvents: 'none' }}>
            <Icon name="filter" size={13} />
          </span>
          <input className="input" placeholder="Search artifacts…" value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ paddingLeft: 32, height: 32, fontSize: 'var(--text-sm)' }} />
        </div>
        <select value={filterType} onChange={e => setFilterType(e.target.value)}
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--r-md)', color: 'var(--text-muted)', fontSize: 'var(--text-xs)',
            padding: '5px 10px', fontFamily: 'var(--font-mono)', cursor: 'pointer', height: 32 }}>
          <option value="all">All types</option>
          {allTypes.map(t => <option key={t} value={t}>{t}</option>)}
        </select>
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--r-md)', color: 'var(--text-muted)', fontSize: 'var(--text-xs)',
            padding: '5px 10px', fontFamily: 'var(--font-mono)', cursor: 'pointer', height: 32 }}>
          <option value="all">All statuses</option>
          {allStatuses.map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <button className="btn btn-primary btn-sm">
          <Icon name="plus" size={12} /> New
        </button>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {filtered.length} / {artifacts.length}
        </span>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead style={{ position: 'sticky', top: 0, background: 'var(--bg-raised)', zIndex: 1 }}>
            <tr>
              <SortHeader label="Title"   k="title" />
              <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)',
                fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Type</th>
              <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)',
                fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Status</th>
              <SortHeader label="Lineage" k="lineage" />
              <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)',
                fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Labels</th>
              <SortHeader label="Updated" k="updatedAt" />
              <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)', width: 60 }}></th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((a, i) => (
              <tr key={a.path}
                onClick={() => onSelect(a)}
                style={{
                  background: i % 2 === 0 ? 'transparent' : 'var(--bg-raised)',
                  cursor: 'pointer', transition: 'background var(--dur-fast)',
                }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-elevated)'}
                onMouseLeave={e => e.currentTarget.style.background = i % 2 === 0 ? 'transparent' : 'var(--bg-raised)'}
              >
                <td style={{ padding: '10px 12px', maxWidth: 280 }}>
                  <div style={{ fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {a.title}
                  </div>
                  <div style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)',
                    fontFamily: 'var(--font-mono)', marginTop: 2 }}>
                    {a.path}
                  </div>
                </td>
                <td style={{ padding: '10px 12px', whiteSpace: 'nowrap' }}>
                  <TypeBadge type={a.type} />
                </td>
                <td style={{ padding: '10px 12px', whiteSpace: 'nowrap' }}>
                  <StatusBadge status={a.status} />
                </td>
                <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
                  {a.lineage}
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <div style={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
                    {(a.labels || []).map(l => <LabelChip key={l} label={l} />)}
                  </div>
                </td>
                <td style={{ padding: '10px 12px', whiteSpace: 'nowrap',
                  fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
                  {a.updatedAt ? new Date(a.updatedAt).toLocaleDateString() : '—'}
                </td>
                <td style={{ padding: '10px 6px' }}>
                  <button className="btn btn-ghost btn-icon btn-sm" onClick={e => { e.stopPropagation(); onEdit(a); }}>
                    <Icon name="edit" size={12} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {filtered.length === 0 && <EmptyState icon="list" title="No artifacts match" subtitle="Try adjusting your filters" />}
      </div>
    </div>
  );
}

// ── Artifact Editor ──
function ArtifactEditorView({ artifact, onClose, onSave, addToast }) {
  const [content, setContent] = React.useState('');
  const [preview, setPreview] = React.useState(true);
  const [saved, setSaved] = React.useState(false);
  const [showFrontmatter, setShowFrontmatter] = React.useState(true);
  const textareaRef = React.useRef(null);

  React.useEffect(() => {
    if (!artifact) return;
    const fm = [
      '---',
      `title: ${artifact.title}`,
      `type: ${artifact.type}`,
      `status: ${artifact.status}`,
      `lineage: ${artifact.lineage}`,
      artifact.parent ? `parent: ${artifact.parent}` : null,
      artifact.labels?.length ? `labels: [${artifact.labels.join(', ')}]` : null,
      artifact.release ? `release: ${artifact.release}` : null,
      artifact.sprint  ? `sprint: ${artifact.sprint}` : null,
      '---',
      '',
      artifact.body || '',
    ].filter(l => l !== null).join('\n');
    setContent(fm);
  }, [artifact]);

  function handleSave() {
    setSaved(true);
    addToast('Artifact saved and committed', 'success');
    setTimeout(() => setSaved(false), 2000);
  }

  function handleKeyDown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      handleSave();
    }
    if (e.key === 'Tab') {
      e.preventDefault();
      const ta = textareaRef.current;
      const start = ta.selectionStart, end = ta.selectionEnd;
      const newVal = content.slice(0, start) + '  ' + content.slice(end);
      setContent(newVal);
      setTimeout(() => { ta.selectionStart = ta.selectionEnd = start + 2; }, 0);
    }
  }

  const renderPreview = (text) => {
    const lines = text.split('\n');
    let inFm = false, fmDone = false, elements = [];
    lines.forEach((line, i) => {
      if (i === 0 && line === '---') { inFm = true; return; }
      if (inFm && line === '---') { inFm = false; fmDone = true; return; }
      if (inFm) return;
      if (line.startsWith('# '))  { elements.push(<h1 key={i} style={{ fontSize: 22, fontWeight: 800, marginBottom: 8, marginTop: 16 }}>{line.slice(2)}</h1>); return; }
      if (line.startsWith('## ')) { elements.push(<h2 key={i} style={{ fontSize: 17, fontWeight: 700, marginBottom: 6, marginTop: 20, color: 'var(--text)' }}>{line.slice(3)}</h2>); return; }
      if (line.startsWith('### ')){ elements.push(<h3 key={i} style={{ fontSize: 14, fontWeight: 600, marginBottom: 4, marginTop: 14, color: 'var(--text-muted)' }}>{line.slice(4)}</h3>); return; }
      if (line.startsWith('- '))  { elements.push(<li key={i} style={{ marginLeft: 18, marginBottom: 3, color: 'var(--text-muted)' }}>{line.slice(2)}</li>); return; }
      if (line.startsWith('`') && line.endsWith('`')) {
        elements.push(<code key={i} style={{ display: 'block', fontFamily: 'var(--font-mono)', background: 'var(--bg-overlay)', padding: '8px 12px', borderRadius: 'var(--r-md)', fontSize: 12, marginBottom: 8 }}>{line.replace(/`/g,'')}</code>);
        return;
      }
      if (line === '') { elements.push(<div key={i} style={{ height: 8 }} />); return; }
      elements.push(<p key={i} style={{ marginBottom: 4, color: 'var(--text)', lineHeight: 1.7 }}>{line}</p>);
    });
    return elements;
  };

  if (!artifact) return (
    <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <EmptyState icon="edit" title="No artifact selected" subtitle="Click an artifact to edit" />
    </div>
  );

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Editor toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 14px',
        borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-raised)',
        flexShrink: 0,
      }}>
        <button className="btn btn-ghost btn-sm" onClick={onClose}><Icon name="arrow" size={12} style={{ transform: 'rotate(180deg)' }} /> Back</button>
        <div style={{ height: 14, width: 1, background: 'var(--border-subtle)' }} />
        <div style={{ fontWeight: 600, fontSize: 'var(--text-sm)', flex: 1 }}>{artifact.title}</div>
        <TypeBadge type={artifact.type} />
        <StatusBadge status={artifact.status} />
        <div style={{ height: 14, width: 1, background: 'var(--border-subtle)' }} />
        <button className="btn btn-ghost btn-sm" onClick={() => setShowFrontmatter(f => !f)}>
          <Icon name="settings" size={12} /> FM
        </button>
        <button className="btn btn-ghost btn-sm" onClick={() => setPreview(p => !p)}>
          <Icon name="eye" size={12} /> {preview ? 'Hide' : 'Show'} preview
        </button>
        <button className={`btn btn-sm ${saved ? 'btn-ghost' : 'btn-primary'}`} onClick={handleSave}>
          {saved ? <><Icon name="check" size={12} /> Saved</> : <><Icon name="copy" size={12} /> Save</>}
        </button>
      </div>

      {/* Lock banner (simulated) */}
      <div style={{
        padding: '6px 14px', background: 'oklch(0.78 0.17 82 / 0.1)', borderBottom: '1px solid oklch(0.78 0.17 82 / 0.3)',
        display: 'flex', alignItems: 'center', gap: 8, fontSize: 'var(--text-xs)',
        color: 'oklch(0.78 0.17 82)',
      }}>
        <Icon name="lock" size={12} /> You hold the lock on lineage <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{artifact.lineage}</span>
        <span style={{ color: 'var(--text-faint)' }}>· Released on close · Heartbeat in 28s</span>
      </div>

      {/* Panes */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Frontmatter panel */}
        {showFrontmatter && (
          <div style={{
            width: 240, flexShrink: 0, borderRight: '1px solid var(--border-subtle)',
            overflow: 'auto', padding: '14px 14px', background: 'var(--bg-raised)',
          }}>
            <div style={{ fontSize: 10, fontWeight: 700, letterSpacing: '0.08em',
              textTransform: 'uppercase', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', marginBottom: 12 }}>
              Frontmatter
            </div>
            {[
              { key: 'title',   label: 'Title',   value: artifact.title },
              { key: 'type',    label: 'Type',    value: artifact.type },
              { key: 'status',  label: 'Status',  value: artifact.status },
              { key: 'lineage', label: 'Lineage', value: artifact.lineage },
              { key: 'release', label: 'Release', value: artifact.release || '' },
              { key: 'sprint',  label: 'Sprint',  value: artifact.sprint || '' },
            ].map(f => (
              <div key={f.key} style={{ marginBottom: 12 }}>
                <label style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                  display: 'block', marginBottom: 4, textTransform: 'lowercase' }}>{f.label}</label>
                <input className="input" defaultValue={f.value}
                  style={{ fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', padding: '4px 8px', height: 28 }} />
              </div>
            ))}
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                display: 'block', marginBottom: 4 }}>labels</label>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {(artifact.labels || []).map(l => <LabelChip key={l} label={l} />)}
                <button className="btn btn-ghost btn-sm" style={{ padding: '1px 6px', fontSize: 10 }}>+ add</button>
              </div>
            </div>
          </div>
        )}

        {/* Editor */}
        <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden',
            borderRight: preview ? '1px solid var(--border-subtle)' : 'none' }}>
            {/* Line numbers + editor */}
            <div style={{ flex: 1, overflow: 'auto', display: 'flex', background: 'var(--bg)' }}>
              <div style={{
                padding: '16px 8px', background: 'var(--bg-raised)', borderRight: '1px solid var(--border-subtle)',
                userSelect: 'none', color: 'var(--text-faint)', fontSize: 11,
                fontFamily: 'var(--font-mono)', lineHeight: '1.6', textAlign: 'right', minWidth: 40,
              }}>
                {content.split('\n').map((_, i) => <div key={i}>{i + 1}</div>)}
              </div>
              <textarea
                ref={textareaRef}
                value={content}
                onChange={e => setContent(e.target.value)}
                onKeyDown={handleKeyDown}
                spellCheck={false}
                style={{
                  flex: 1, padding: '16px 14px', background: 'transparent',
                  border: 'none', outline: 'none', resize: 'none',
                  fontFamily: 'var(--font-mono)', fontSize: 12, lineHeight: '1.6',
                  color: 'var(--text)', width: '100%',
                }}
              />
            </div>
            <div style={{
              padding: '4px 12px', borderTop: '1px solid var(--border-subtle)',
              background: 'var(--bg-raised)', display: 'flex', gap: 16,
              fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
            }}>
              <span>{content.split('\n').length} lines</span>
              <span>{content.length} chars</span>
              <span>Markdown · YAML frontmatter</span>
              <span style={{ flex: 1 }} />
              <span>⌘S save · ⌘K wiki-link</span>
            </div>
          </div>

          {/* Preview */}
          {preview && (
            <div style={{
              flex: 1, overflow: 'auto', padding: '20px 24px',
              background: 'var(--bg)', lineHeight: 1.7,
            }}>
              <div style={{ fontSize: 10, fontWeight: 700, letterSpacing: '0.08em',
                textTransform: 'uppercase', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', marginBottom: 16 }}>
                Preview
              </div>
              {renderPreview(content)}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Agent Runs ──
function AgentsRunsView({ agents, onKill, addToast }) {
  const [runAgentOpen, setRunAgentOpen] = React.useState(false);
  const [expandedRow, setExpandedRow] = React.useState(null);
  const [liveAgents, setLiveAgents] = React.useState(agents);

  // Simulate progress
  React.useEffect(() => {
    const interval = setInterval(() => {
      setLiveAgents(prev => prev.map(a => {
        if (a.status !== 'running') return a;
        const newPct = Math.min(100, (a.pct || 0) + (Math.random() * 3));
        if (newPct >= 100) return { ...a, pct: 100, status: 'finished', finishedAt: new Date().toISOString(), artifactsProduced: [a.target] };
        const messages = [
          'Reading parent artifact…', 'Analysing requirements…', 'Generating plan structure…',
          'Writing section: Overview', 'Writing section: Components', 'Writing section: API contracts',
          'Running validation checks…', 'Committing artifact…',
        ];
        return { ...a, pct: newPct, lastMessage: messages[Math.floor(newPct / 14)] };
      }));
    }, 800);
    return () => clearInterval(interval);
  }, []);

  const statusColor = { running: 'var(--status-in-development)', finished: 'var(--status-done)', crashed: 'oklch(0.60 0.22 22)', killed: 'var(--status-abandoned)' };
  const statusIcon  = { running: 'play', finished: 'check', crashed: 'warning', killed: 'stop' };

  function elapsed(start, end) {
    const s = new Date(start), e = end ? new Date(end) : new Date();
    const ms = e - s;
    const m = Math.floor(ms / 60000), sec = Math.floor((ms % 60000) / 1000);
    return m > 0 ? `${m}m ${sec}s` : `${sec}s`;
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px',
        borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-raised)', flexShrink: 0,
      }}>
        <span style={{ fontWeight: 600, fontSize: 'var(--text-md)' }}>Agent Runs</span>
        <div style={{ flex: 1 }} />
        <button className="btn btn-primary btn-sm" onClick={() => setRunAgentOpen(true)}>
          <Icon name="bot" size={12} /> Run Agent
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead style={{ position: 'sticky', top: 0, background: 'var(--bg-raised)', zIndex: 1 }}>
            <tr>
              {['Status','Agent','Role','Target','Started','Elapsed','Actions'].map(h => (
                <th key={h} style={{
                  padding: '8px 12px', textAlign: 'left', borderBottom: '1px solid var(--border-subtle)',
                  fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', color: 'var(--text-faint)',
                  fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em', whiteSpace: 'nowrap',
                }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {liveAgents.map((run, i) => (
              <React.Fragment key={run.id}>
                <tr onClick={() => setExpandedRow(expandedRow === run.id ? null : run.id)}
                  style={{
                    cursor: 'pointer', transition: 'background var(--dur-fast)',
                    background: expandedRow === run.id ? 'var(--bg-elevated)' : i % 2 === 0 ? 'transparent' : 'var(--bg-raised)',
                  }}
                  onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-elevated)'}
                  onMouseLeave={e => e.currentTarget.style.background = expandedRow === run.id ? 'var(--bg-elevated)' : i % 2 === 0 ? 'transparent' : 'var(--bg-raised)'}
                >
                  <td style={{ padding: '10px 12px', whiteSpace: 'nowrap' }}>
                    <span style={{
                      display: 'inline-flex', alignItems: 'center', gap: 5,
                      color: statusColor[run.status], fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)',
                    }}>
                      {run.status === 'running' ? <Spinner size={11} /> : <Icon name={statusIcon[run.status]} size={11} />}
                      {run.status}
                    </span>
                  </td>
                  <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)', color: 'var(--accent)' }}>{run.agent}</td>
                  <td style={{ padding: '10px 12px' }}><LabelChip label={run.role} /></td>
                  <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)', color: 'var(--text-muted)', maxWidth: 200 }}>
                    <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{run.target}</div>
                  </td>
                  <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)', color: 'var(--text-faint)', whiteSpace: 'nowrap' }}>
                    {new Date(run.startedAt).toLocaleTimeString()}
                  </td>
                  <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
                    {elapsed(run.startedAt, run.finishedAt)}
                  </td>
                  <td style={{ padding: '10px 6px' }}>
                    {run.status === 'running' && (
                      <button className="btn btn-danger btn-sm" onClick={e => { e.stopPropagation(); addToast(`Killed run ${run.id}`, 'info'); setLiveAgents(prev => prev.map(a => a.id === run.id ? { ...a, status: 'killed', finishedAt: new Date().toISOString() } : a)); }}>
                        <Icon name="kill" size={11} /> Kill
                      </button>
                    )}
                  </td>
                </tr>
                {expandedRow === run.id && (
                  <tr style={{ background: 'var(--bg-elevated)' }}>
                    <td colSpan={7} style={{ padding: '0 12px 14px 12px' }}>
                      {run.status === 'running' && (
                        <div style={{ marginBottom: 10 }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6,
                            fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
                            <span>{run.lastMessage}</span>
                            <span>{Math.round(run.pct)}%</span>
                          </div>
                          <ProgressBar pct={run.pct} />
                        </div>
                      )}
                      <div style={{
                        background: 'var(--bg)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--r-md)',
                        padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)',
                        maxHeight: 120, overflow: 'auto', lineHeight: 1.6,
                      }}>
                        {run.status === 'running' && `[${new Date().toLocaleTimeString()}] ${run.lastMessage || 'Agent running…'}\n[${new Date(run.startedAt).toLocaleTimeString()}] Agent started\n[${new Date(run.startedAt).toLocaleTimeString()}] Reading target: ${run.target}`}
                        {run.status === 'finished' && `[${new Date(run.finishedAt).toLocaleTimeString()}] Finished. Artifacts: ${run.artifactsProduced?.join(', ')}\n[${new Date(run.startedAt).toLocaleTimeString()}] Agent started`}
                        {run.status === 'crashed' && `[error] ${run.stderrTail}\n[exit code] ${run.exitCode}\n[${new Date(run.startedAt).toLocaleTimeString()}] Agent started`}
                        {run.status === 'killed' && `[killed] Process terminated by user`}
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            ))}
          </tbody>
        </table>
      </div>

      {runAgentOpen && <RunAgentDialog onClose={() => setRunAgentOpen(false)} addToast={addToast} />}
    </div>
  );
}

function RunAgentDialog({ onClose, addToast }) {
  const [selectedAgent, setSelectedAgent] = React.useState('claude-planner');
  const [targetPath, setTargetPath] = React.useState('requirements/password-reset-2.md');
  const [showPrompt, setShowPrompt] = React.useState(false);

  const promptPreview = `You are a frontend-planner agent. Your task is to produce a frontend plan artifact for the following requirement:\n\nArtifact: lifecycle/${targetPath}\n\nRead the artifact, understand the requirements, and produce a comprehensive frontend plan in markdown format covering:\n1. Components needed\n2. State management\n3. API contracts\n4. UX behaviour\n\nWrite the output to: lifecycle/frontend-plans/<slug>-<next-index>.md\nCommit with your configured git identity.`;

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 3000,
      background: 'oklch(0 0 0 / 0.6)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
    }} onClick={onClose}>
      <div onClick={e => e.stopPropagation()} style={{
        background: 'var(--bg-elevated)', border: '1px solid var(--border)',
        borderRadius: 'var(--r-xl)', width: '100%', maxWidth: 520,
        boxShadow: 'var(--shadow)', overflow: 'hidden',
      }}>
        <div style={{ padding: '18px 20px', borderBottom: '1px solid var(--border-subtle)', display: 'flex', alignItems: 'center', gap: 8 }}>
          <Icon name="bot" size={16} style={{ color: 'var(--accent)' }} />
          <span style={{ fontWeight: 700, fontSize: 'var(--text-lg)' }}>Run Agent</span>
          <div style={{ flex: 1 }} />
          <button className="btn btn-ghost btn-icon btn-sm" onClick={onClose}><Icon name="close" size={13} /></button>
        </div>
        <div style={{ padding: '20px' }}>
          <div style={{ marginBottom: 16 }}>
            <label style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', display: 'block', marginBottom: 6 }}>Agent</label>
            <select value={selectedAgent} onChange={e => setSelectedAgent(e.target.value)}
              style={{ width: '100%', background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--r-md)', color: 'var(--text)', fontSize: 'var(--text-sm)',
                padding: '8px 12px', fontFamily: 'var(--font-mono)', cursor: 'pointer' }}>
              <option value="claude-planner">claude-planner (backend-planner, frontend-planner)</option>
              <option value="local-dev">local-dev (developer)</option>
            </select>
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', display: 'block', marginBottom: 6 }}>Target artifact</label>
            <input className="input" value={targetPath} onChange={e => setTargetPath(e.target.value)}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }} />
          </div>
          <button className="btn btn-ghost btn-sm" onClick={() => setShowPrompt(p => !p)} style={{ marginBottom: showPrompt ? 10 : 0 }}>
            <Icon name="eye" size={12} /> {showPrompt ? 'Hide' : 'Preview'} prompt
          </button>
          {showPrompt && (
            <div style={{
              background: 'var(--bg)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--r-md)',
              padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: 11,
              color: 'var(--text-muted)', lineHeight: 1.6, maxHeight: 160, overflow: 'auto', marginBottom: 16,
              whiteSpace: 'pre-wrap',
            }}>{promptPreview}</div>
          )}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 20 }}>
            <button className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
            <button className="btn btn-primary btn-sm" onClick={() => {
              addToast(`Started ${selectedAgent} on ${targetPath}`, 'success');
              onClose();
            }}>
              <Icon name="play" size={12} /> Confirm &amp; Run
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Project Config View ──
function ProjectConfigView({ addToast }) {
  const [tab, setTab] = React.useState('stages');
  const stages = ['ideas','requirements','backend-plans','frontend-plans','dev-plans','test-plans','tests','prototypes','releases','sprints'];
  const agents = [
    { name: 'claude-planner', roles: ['backend-planner','frontend-planner'], driver: 'claude-code-cli', model: 'claude-sonnet-4-6' },
    { name: 'local-dev',      roles: ['developer'],                          driver: 'mcp',            endpoint: 'http://localhost:3210' },
  ];

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-raised)', flexShrink: 0 }}>
        <span style={{ fontWeight: 600, fontSize: 'var(--text-md)' }}>Project Config</span>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', marginLeft: 8 }}>lifecycle/config.yaml</span>
      </div>
      <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid var(--border-subtle)', padding: '0 16px', background: 'var(--bg-raised)', flexShrink: 0 }}>
        {['stages','agents','roles','git','raw yaml'].map(t => (
          <button key={t} onClick={() => setTab(t)} style={{
            background: 'none', border: 'none', cursor: 'pointer', padding: '10px 14px',
            fontSize: 'var(--text-sm)', fontWeight: tab === t ? 600 : 400,
            color: tab === t ? 'var(--accent)' : 'var(--text-muted)',
            borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
            marginBottom: -1, textTransform: 'capitalize',
          }}>{t}</button>
        ))}
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px' }}>
        {tab === 'stages' && (
          <div>
            <div style={{ marginBottom: 16, color: 'var(--text-muted)', fontSize: 'var(--text-sm)' }}>Lifecycle stages (drag to reorder)</div>
            {stages.map((s, i) => (
              <div key={s} style={{
                display: 'flex', alignItems: 'center', gap: 12, padding: '10px 14px',
                background: 'var(--bg-raised)', border: '1px solid var(--border-subtle)',
                borderRadius: 'var(--r-md)', marginBottom: 6, cursor: 'grab',
              }}>
                <span style={{ color: 'var(--text-faint)', fontSize: 12 }}>⋮⋮</span>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--text-sm)', flex: 1 }}>{s}</span>
                <input className="input" defaultValue={s} style={{ width: 160, fontSize: 'var(--text-xs)', fontFamily: 'var(--font-mono)', padding: '4px 8px' }} />
              </div>
            ))}
          </div>
        )}
        {tab === 'agents' && (
          <div>
            {agents.map(a => (
              <div key={a.name} className="card" style={{ padding: '16px', marginBottom: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
                  <Icon name="bot" size={15} style={{ color: 'var(--accent)' }} />
                  <span style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>{a.name}</span>
                  <LabelChip label={a.driver} />
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '120px 1fr', gap: '8px 12px', fontSize: 'var(--text-sm)' }}>
                  <span style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>roles:</span>
                  <span>{a.roles.map(r => <LabelChip key={r} label={r} />)}</span>
                  <span style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>driver:</span>
                  <span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>{a.driver}</span>
                  {a.model && <><span style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>model:</span><span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>{a.model}</span></>}
                  {a.endpoint && <><span style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>endpoint:</span><span style={{ fontFamily: 'var(--font-mono)', fontSize: 'var(--text-xs)' }}>{a.endpoint}</span></>}
                </div>
              </div>
            ))}
            <button className="btn btn-ghost btn-sm"><Icon name="plus" size={12} /> Add agent</button>
          </div>
        )}
        {tab === 'raw yaml' && (
          <div style={{ background: 'var(--bg)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--r-md)', padding: '14px', fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)', lineHeight: 1.7, whiteSpace: 'pre', overflow: 'auto' }}>{`stages:
  - name: ideas
    dir: ideas
  - name: requirements
    dir: requirements
  - name: backend-plans
    dir: backend-plans
  - name: frontend-plans
    dir: frontend-plans
  - name: dev-plans
    dir: dev-plans
  - name: releases
    dir: releases
  - name: sprints
    dir: sprints

git:
  default_branch: main
  branch_template: "ticket/{slug}"

agents:
  - name: claude-planner
    role: [backend-planner, frontend-planner]
    driver: claude-code-cli
    model: claude-sonnet-4-6
    git_identity:
      name: claude-planner
      email: planner@innovation-maker.local
  - name: local-dev
    role: [developer]
    driver: mcp
    endpoint: http://localhost:3210

users:
  - email: keith@sinclair.org.au
    roles: [product-owner, approver]`}</div>
        )}
      </div>
    </div>
  );
}

// ── Login View ──
function LoginView({ onLogin }) {
  const [email, setEmail] = React.useState('keith@sinclair.org.au');
  const [password, setPassword] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState('');

  function handleSubmit(e) {
    e.preventDefault();
    if (!email || !password) { setError('Email and password required.'); return; }
    setLoading(true); setError('');
    setTimeout(() => { setLoading(false); onLogin(email); }, 1000);
  }

  return (
    <div style={{
      height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'var(--bg)', padding: 24,
    }}>
      <div style={{ width: '100%', maxWidth: 380 }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: 40 }}>
          <div style={{
            width: 56, height: 56, borderRadius: 'var(--r-xl)',
            background: 'var(--accent)', margin: '0 auto 16px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 0 32px var(--accent-glow)',
          }}>
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/>
              <line x1="12" y1="22" x2="12" y2="15.5"/>
              <polyline points="22 8.5 12 15.5 2 8.5"/>
            </svg>
          </div>
          <div style={{ fontSize: 'var(--text-xl)', fontWeight: 800, letterSpacing: '-0.02em' }}>
            kaos<span style={{ color: 'var(--accent)' }}>-</span>control
          </div>
          <div style={{ color: 'var(--text-faint)', fontSize: 'var(--text-sm)', marginTop: 4 }}>Sign in to continue</div>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div>
            <label style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', display: 'block', marginBottom: 5 }}>email</label>
            <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} autoComplete="email" />
          </div>
          <div>
            <label style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', display: 'block', marginBottom: 5 }}>password</label>
            <input className="input" type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="••••••••" autoComplete="current-password" />
          </div>
          {error && <div style={{ color: 'oklch(0.65 0.22 22)', fontSize: 'var(--text-xs)', display: 'flex', gap: 5, alignItems: 'center' }}><Icon name="warning" size={12} />{error}</div>}
          <button type="submit" className="btn btn-primary" style={{ justifyContent: 'center', marginTop: 4 }} disabled={loading}>
            {loading ? <><Spinner size={13} /> Signing in…</> : 'Sign in →'}
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: 24, fontSize: 'var(--text-xs)', color: 'var(--text-faint)' }}>
          Local account · v1 · kaos-control
        </div>
      </div>
    </div>
  );
}

// ── Project Picker ──
function ProjectPickerView({ projects, onSelect, onCreateProject, userEmail }) {
  const [showCreate, setShowCreate] = React.useState(false);
  const [newName, setNewName] = React.useState('');
  const [newPath, setNewPath] = React.useState('/home/keith/Projects/');
  const [newDesc, setNewDesc] = React.useState('');

  const typeColors = { 'idea': '#e8b94d', 'ticket': '#5eb8f7', 'epic': '#d462f7', 'plan-backend': '#8b72f7', 'plan-frontend': '#f76aae', 'release': '#f79a3a' };

  return (
    <div style={{ height: '100%', overflow: 'auto', padding: '40px 32px', background: 'var(--bg)' }}>
      <div style={{ maxWidth: 780, margin: '0 auto' }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 32 }}>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 'var(--text-2xl)', fontWeight: 800, letterSpacing: '-0.02em', marginBottom: 4 }}>Projects</div>
            <div style={{ color: 'var(--text-muted)', fontSize: 'var(--text-sm)' }}>Signed in as <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)' }}>{userEmail}</span></div>
          </div>
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            <Icon name="plus" size={14} /> New project
          </button>
        </div>

        {/* Project cards */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
          {projects.map(p => (
            <div key={p.name} className="card" onClick={() => onSelect(p.name)}
              style={{ padding: '20px', cursor: 'pointer', transition: 'all var(--dur-fast) var(--ease)' }}>
              <div style={{ display: 'flex', align: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
                <div style={{
                  width: 36, height: 36, borderRadius: 'var(--r-md)', display: 'flex', alignItems: 'center', justifyContent: 'center',
                  background: 'color-mix(in oklch, var(--accent) 15%, transparent)', color: 'var(--accent)',
                }}>
                  <Icon name="folder" size={18} />
                </div>
                <span style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
                  alignSelf: 'flex-start', marginTop: 6 }}>{p.artifactCount} artifacts</span>
              </div>
              <div style={{ fontWeight: 700, fontSize: 'var(--text-md)', marginBottom: 4, fontFamily: 'var(--font-mono)' }}>{p.name}</div>
              <div style={{ color: 'var(--text-muted)', fontSize: 'var(--text-sm)', marginBottom: 14 }}>{p.description}</div>
              <div style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)',
                overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.path}</div>
            </div>
          ))}
        </div>

        {showCreate && (
          <div style={{
            position: 'fixed', inset: 0, zIndex: 3000,
            background: 'oklch(0 0 0 / 0.6)', backdropFilter: 'blur(4px)',
            display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
          }} onClick={() => setShowCreate(false)}>
            <div onClick={e => e.stopPropagation()} style={{
              background: 'var(--bg-elevated)', border: '1px solid var(--border)',
              borderRadius: 'var(--r-xl)', width: '100%', maxWidth: 440, padding: '24px',
              boxShadow: 'var(--shadow)',
            }}>
              <div style={{ fontWeight: 700, fontSize: 'var(--text-lg)', marginBottom: 20 }}>Register project</div>
              {[{ label: 'name', val: newName, set: setNewName, placeholder: 'my-project' },
                { label: 'path', val: newPath, set: setNewPath, placeholder: '/home/keith/Projects/my-project' },
                { label: 'description', val: newDesc, set: setNewDesc, placeholder: 'What is this project?' }
              ].map(f => (
                <div key={f.label} style={{ marginBottom: 14 }}>
                  <label style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', display: 'block', marginBottom: 5 }}>{f.label}</label>
                  <input className="input" value={f.val} onChange={e => f.set(e.target.value)} placeholder={f.placeholder} style={{ fontFamily: f.label !== 'description' ? 'var(--font-mono)' : undefined, fontSize: 'var(--text-sm)' }} />
                </div>
              ))}
              <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 20 }}>
                <button className="btn btn-ghost btn-sm" onClick={() => setShowCreate(false)}>Cancel</button>
                <button className="btn btn-primary btn-sm" onClick={() => { onCreateProject({ name: newName, path: newPath, description: newDesc }); setShowCreate(false); }}>
                  Register
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

Object.assign(window, {
  ArtifactListView, ArtifactEditorView, AgentsRunsView, RunAgentDialog,
  ProjectConfigView, LoginView, ProjectPickerView,
});
