// ── kc-graph.jsx — Canvas force graph + Artifact Modal ──

// ── Force simulation ──
function createForceGraph(nodes, links, width, height) {
  const state = {
    nodes: nodes.map((n, i) => ({
      ...n,
      x: width / 2 + (Math.cos(i / nodes.length * Math.PI * 2) * 200),
      y: height / 2 + (Math.sin(i / nodes.length * Math.PI * 2) * 200),
      vx: 0, vy: 0,
    })),
    links,
    running: true,
  };

  function tick() {
    if (!state.running) return;
    const ns = state.nodes;
    const cx = width / 2, cy = height / 2;
    const repulse = 3200, attract = 0.04, center = 0.012, dampen = 0.82;

    // Center force
    ns.forEach(n => {
      n.vx += (cx - n.x) * center;
      n.vy += (cy - n.y) * center;
    });

    // Repulsion
    for (let i = 0; i < ns.length; i++) {
      for (let j = i + 1; j < ns.length; j++) {
        const dx = ns[j].x - ns[i].x, dy = ns[j].y - ns[i].y;
        const d2 = dx * dx + dy * dy + 1;
        const f = repulse / d2;
        ns[i].vx -= dx * f; ns[i].vy -= dy * f;
        ns[j].vx += dx * f; ns[j].vy += dy * f;
      }
    }

    // Link attraction
    state.links.forEach(l => {
      const s = ns.find(n => n.id === l.source);
      const t = ns.find(n => n.id === l.target);
      if (!s || !t) return;
      const dx = t.x - s.x, dy = t.y - s.y;
      s.vx += dx * attract; s.vy += dy * attract;
      t.vx -= dx * attract; t.vy -= dy * attract;
    });

    // Integrate
    ns.forEach(n => {
      n.vx *= dampen; n.vy *= dampen;
      n.x = Math.max(60, Math.min(width - 60, n.x + n.vx));
      n.y = Math.max(60, Math.min(height - 60, n.y + n.vy));
    });
  }

  return { state, tick };
}

// ── Main Graph Component ──
function GraphView({ artifacts, links, onNodeClick, selectedNode, glowStyle }) {
  const canvasRef = React.useRef(null);
  const simRef = React.useRef(null);
  const rafRef = React.useRef(null);
  const [hoveredNode, setHoveredNode] = React.useState(null);
  const [filterType, setFilterType] = React.useState('all');
  const [filterStatus, setFilterStatus] = React.useState('all');
  const [showLegend, setShowLegend] = React.useState(true);
  const [tooltip, setTooltip] = React.useState(null);
  const [camera, setCamera] = React.useState({ x: 0, y: 0, scale: 1 });
  const panRef = React.useRef(null);
  const cameraRef = React.useRef(camera);
  React.useEffect(() => { cameraRef.current = camera; }, [camera]);

  const typeColors = {
    'idea':          '#e8b94d', 'ticket':        '#5eb8f7', 'epic':          '#d462f7',
    'plan-backend':  '#8b72f7', 'plan-frontend': '#f76aae', 'plan-dev':      '#5fd482',
    'plan-test':     '#42cca8', 'release':       '#f79a3a', 'sprint':        '#38c6e8',
    'test':          '#52c99a', 'prototype':     '#f76e42',
  };
  const edgeColors = {
    'parent': '#7264e8', 'depends_on': '#e8a030', 'blocks': '#e85030',
    'member_of': '#38c6e8', 'related_to': '#666',
  };

  const filteredNodes = React.useMemo(() => {
    return artifacts.filter(a =>
      (filterType === 'all' || a.type === filterType) &&
      (filterStatus === 'all' || a.status === filterStatus)
    );
  }, [artifacts, filterType, filterStatus]);

  const filteredNodeIds = React.useMemo(() => new Set(filteredNodes.map(a => a.path)), [filteredNodes]);

  const filteredLinks = React.useMemo(() =>
    links.filter(l => filteredNodeIds.has(l.source) && filteredNodeIds.has(l.target)),
    [links, filteredNodeIds]
  );

  // Build sim nodes/links
  const simNodes = filteredNodes.map(a => ({ id: a.path, type: a.type, title: a.title, status: a.status }));
  const simLinks = filteredLinks.map(l => ({ source: l.source, target: l.target, kind: l.kind }));

  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const { width, height } = canvas;
    simRef.current = createForceGraph(simNodes, simLinks, width, height);

    let frame = 0;
    function loop() {
      if (!simRef.current) return;
      simRef.current.tick();
      drawGraph();
      frame++;
      // Slow down after settling
      const interval = frame < 150 ? 16 : frame < 300 ? 32 : 60;
      rafRef.current = setTimeout(loop, interval);
    }
    rafRef.current = setTimeout(loop, 16);
    return () => { clearTimeout(rafRef.current); simRef.current = null; };
  }, [filteredNodes.length, filteredLinks.length]);

  function drawGraph() {
    const canvas = canvasRef.current;
    if (!canvas || !simRef.current) return;
    const ctx = canvas.getContext('2d');
    const { width, height } = canvas;
    const { state } = simRef.current;
    const { x: cx, y: cy, scale } = cameraRef.current;

    ctx.clearRect(0, 0, width, height);

    ctx.save();
    ctx.translate(cx, cy);
    ctx.scale(scale, scale);

    // Draw edges
    state.links.forEach(l => {
      const s = state.nodes.find(n => n.id === l.source);
      const t = state.nodes.find(n => n.id === l.target);
      if (!s || !t) return;
      const color = edgeColors[l.kind] || '#555';

      ctx.beginPath();
      ctx.moveTo(s.x, s.y);
      ctx.lineTo(t.x, t.y);
      ctx.strokeStyle = color + '88';
      ctx.lineWidth = 1.5;
      ctx.setLineDash(l.kind === 'depends_on' ? [4, 3] : []);
      ctx.stroke();
      ctx.setLineDash([]);

      // Arrowhead
      const dx = t.x - s.x, dy = t.y - s.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < 1) return;
      const nx = dx / dist, ny = dy / dist;
      const nodeR = getNodeRadius(t);
      const ax = t.x - nx * (nodeR + 4), ay = t.y - ny * (nodeR + 4);
      const angle = Math.atan2(dy, dx);
      ctx.beginPath();
      ctx.moveTo(ax, ay);
      ctx.lineTo(ax - 8 * Math.cos(angle - 0.4), ay - 8 * Math.sin(angle - 0.4));
      ctx.lineTo(ax - 8 * Math.cos(angle + 0.4), ay - 8 * Math.sin(angle + 0.4));
      ctx.closePath();
      ctx.fillStyle = color + 'aa';
      ctx.fill();
    });

    // Draw nodes
    state.nodes.forEach(n => {
      const color = typeColors[n.type] || '#888';
      const r = getNodeRadius(n);
      const isHovered = hoveredNode === n.id;
      const isSelected = selectedNode && selectedNode.path === n.id;

      if (glowStyle && (isHovered || isSelected)) {
        ctx.beginPath();
        ctx.arc(n.x, n.y, r + 8, 0, Math.PI * 2);
        const grd = ctx.createRadialGradient(n.x, n.y, r, n.x, n.y, r + 16);
        grd.addColorStop(0, color + '55');
        grd.addColorStop(1, color + '00');
        ctx.fillStyle = grd;
        ctx.fill();
      }

      ctx.beginPath();
      ctx.arc(n.x, n.y, r, 0, Math.PI * 2);

      if (isSelected) {
        ctx.strokeStyle = color;
        ctx.lineWidth = 3;
        ctx.stroke();
        ctx.fillStyle = color + '33';
      } else if (isHovered) {
        ctx.strokeStyle = color;
        ctx.lineWidth = 2;
        ctx.stroke();
        ctx.fillStyle = color + '55';
      } else {
        ctx.fillStyle = color;
      }
      ctx.fill();

      // Icon or initial
      ctx.fillStyle = isSelected || isHovered ? color : 'rgba(10,10,20,0.85)';
      ctx.font = `bold ${Math.max(8, r * 0.65)}px JetBrains Mono, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      const initial = (n.type || '?').charAt(0).toUpperCase();
      ctx.fillText(initial, n.x, n.y);

      // Label below
      ctx.fillStyle = isHovered || isSelected ? '#fff' : 'rgba(200,200,220,0.7)';
      ctx.font = `${isHovered ? 600 : 400} 10px Inter, sans-serif`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';
      const label = n.title && n.title.length > 18 ? n.title.slice(0, 17) + '…' : n.title;
      ctx.fillText(label || n.id, n.x, n.y + r + 4);
    });

    ctx.restore();
  }

  function getNodeRadius(n) {
    const base = { idea: 16, ticket: 18, epic: 22, release: 22, sprint: 20 };
    return base[n.type] || 14;
  }

  function getNodeAt(mx, my) {
    if (!simRef.current) return null;
    const { x: cx, y: cy, scale } = cameraRef.current;
    const wx = (mx - cx) / scale, wy = (my - cy) / scale;
    const { state } = simRef.current;
    for (const n of [...state.nodes].reverse()) {
      const dx = n.x - wx, dy = n.y - wy;
      const r = getNodeRadius(n) + 4;
      if (dx * dx + dy * dy < r * r) return n;
    }
    return null;
  }

  function handleMouseMove(e) {
    const rect = canvasRef.current.getBoundingClientRect();
    const mx = e.clientX - rect.left, my = e.clientY - rect.top;
    if (panRef.current) {
      const { startX, startY, startCX, startCY } = panRef.current;
      setCamera(c => ({ ...c, x: startCX + (mx - startX), y: startCY + (my - startY) }));
      return;
    }
    const n = getNodeAt(mx, my);
    setHoveredNode(n ? n.id : null);
    if (n) {
      setTooltip({ x: e.clientX, y: e.clientY, node: n });
      canvasRef.current.style.cursor = 'pointer';
    } else {
      setTooltip(null);
      canvasRef.current.style.cursor = 'grab';
    }
  }

  function handleMouseDown(e) {
    const rect = canvasRef.current.getBoundingClientRect();
    const mx = e.clientX - rect.left, my = e.clientY - rect.top;
    if (!getNodeAt(mx, my)) {
      panRef.current = { startX: mx, startY: my, startCX: cameraRef.current.x, startCY: cameraRef.current.y };
      canvasRef.current.style.cursor = 'grabbing';
    }
  }

  function handleMouseUp() {
    panRef.current = null;
    canvasRef.current.style.cursor = 'grab';
  }

  function handleClick(e) {
    const rect = canvasRef.current.getBoundingClientRect();
    const mx = e.clientX - rect.left, my = e.clientY - rect.top;
    const n = getNodeAt(mx, my);
    if (n) {
      const artifact = artifacts.find(a => a.path === n.id);
      if (artifact) onNodeClick(artifact);
    }
  }

  function handleWheel(e) {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setCamera(c => ({ ...c, scale: Math.max(0.3, Math.min(3, c.scale * delta)) }));
  }

  // Redraw on hover/selected change
  React.useEffect(() => { drawGraph(); }, [hoveredNode, camera]);

  const allTypes = [...new Set(artifacts.map(a => a.type))];
  const allStatuses = [...new Set(artifacts.map(a => a.status))];

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', position: 'relative' }}>
      {/* Toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 14px',
        borderBottom: '1px solid var(--border-subtle)', background: 'var(--bg-raised)',
        flexShrink: 0, flexWrap: 'wrap',
      }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>
          {filteredNodes.length} nodes · {filteredLinks.length} edges
        </span>
        <div style={{ height: 14, width: 1, background: 'var(--border-subtle)' }} />

        <select value={filterType} onChange={e => setFilterType(e.target.value)}
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--r-md)', color: 'var(--text-muted)', fontSize: 'var(--text-xs)',
            padding: '3px 8px', fontFamily: 'var(--font-mono)', cursor: 'pointer' }}>
          <option value="all">All types</option>
          {allTypes.map(t => <option key={t} value={t}>{t}</option>)}
        </select>

        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)}
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--r-md)', color: 'var(--text-muted)', fontSize: 'var(--text-xs)',
            padding: '3px 8px', fontFamily: 'var(--font-mono)', cursor: 'pointer' }}>
          <option value="all">All statuses</option>
          {allStatuses.map(s => <option key={s} value={s}>{s}</option>)}
        </select>

        <div style={{ flex: 1 }} />

        <button className="btn btn-ghost btn-sm" onClick={() => setCamera({ x: 0, y: 0, scale: 1 })}>
          <Icon name="refresh" size={12} /> Reset
        </button>
        <button className="btn btn-ghost btn-sm" onClick={() => setShowLegend(l => !l)}>
          <Icon name="info" size={12} /> Legend
        </button>
      </div>

      {/* Canvas */}
      <div style={{ flex: 1, position: 'relative', background: 'var(--graph-bg)' }}>
        <canvas ref={canvasRef}
          width={1200} height={800}
          style={{ width: '100%', height: '100%', cursor: 'grab', display: 'block' }}
          onMouseMove={handleMouseMove}
          onMouseDown={handleMouseDown}
          onMouseUp={handleMouseUp}
          onMouseLeave={() => { setHoveredNode(null); setTooltip(null); handleMouseUp(); }}
          onClick={handleClick}
          onWheel={handleWheel}
        />

        {/* Tooltip */}
        {tooltip && (
          <div style={{
            position: 'fixed', left: tooltip.x + 12, top: tooltip.y - 10,
            background: 'var(--bg-overlay)', backdropFilter: 'blur(8px)',
            border: '1px solid var(--border)', borderRadius: 'var(--r-md)',
            padding: '8px 12px', fontSize: 'var(--text-xs)', pointerEvents: 'none',
            boxShadow: 'var(--shadow)', zIndex: 500, maxWidth: 220,
          }}>
            <div style={{ fontWeight: 600, marginBottom: 3 }}>{tooltip.node.title}</div>
            <div style={{ color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>{tooltip.node.type} · {tooltip.node.status}</div>
          </div>
        )}

        {/* Legend */}
        {showLegend && (
          <div style={{
            position: 'absolute', bottom: 16, left: 16,
            background: 'var(--bg-overlay)', backdropFilter: 'blur(12px)',
            border: '1px solid var(--border-subtle)', borderRadius: 'var(--r-lg)',
            padding: '12px 14px', fontSize: 'var(--text-xs)', zIndex: 100,
          }}>
            <div style={{ fontWeight: 700, marginBottom: 8, color: 'var(--text-faint)',
              fontFamily: 'var(--font-mono)', textTransform: 'uppercase', letterSpacing: '0.06em', fontSize: 10 }}>
              Node types
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px 16px' }}>
              {Object.entries(typeColors).slice(0, 8).map(([type, color]) => (
                <div key={type} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ width: 10, height: 10, borderRadius: '50%', background: color, flexShrink: 0 }} />
                  <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>{type}</span>
                </div>
              ))}
            </div>
            <div style={{ height: 1, background: 'var(--border-subtle)', margin: '10px 0 8px' }} />
            <div style={{ fontWeight: 700, marginBottom: 6, color: 'var(--text-faint)',
              fontFamily: 'var(--font-mono)', textTransform: 'uppercase', letterSpacing: '0.06em', fontSize: 10 }}>
              Edge types
            </div>
            {Object.entries(edgeColors).map(([kind, color]) => (
              <div key={kind} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 3 }}>
                <span style={{ width: 20, height: 2, background: color, flexShrink: 0, borderRadius: 1 }} />
                <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>{kind}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ── 2D Graph (Cytoscape-style rendered via canvas, simpler layout) ──
function Graph2DView({ artifacts, links, onNodeClick }) {
  const containerRef = React.useRef(null);

  // Hierarchical layout: group by type vertically
  const typeOrder = ['idea','requirements','ticket','epic','plan-backend','plan-frontend','plan-dev','release','sprint'];
  const byType = {};
  artifacts.forEach(a => {
    const key = a.type;
    if (!byType[key]) byType[key] = [];
    byType[key].push(a);
  });

  const positions = {};
  let col = 0;
  Object.entries(byType).forEach(([type, nodes]) => {
    nodes.forEach((n, i) => {
      positions[n.path] = { x: 120 + col * 220, y: 80 + i * 90 };
    });
    col++;
  });

  const typeColors = {
    'idea': '#e8b94d', 'ticket': '#5eb8f7', 'epic': '#d462f7',
    'plan-backend': '#8b72f7', 'plan-frontend': '#f76aae', 'plan-dev': '#5fd482',
    'plan-test': '#42cca8', 'release': '#f79a3a', 'sprint': '#38c6e8',
  };

  const svgW = Math.max(800, col * 220 + 120);
  const svgH = Math.max(600, Math.max(...Object.values(byType).map(g => g.length)) * 90 + 100);

  return (
    <div style={{ flex: 1, overflow: 'auto', background: 'var(--graph-bg)', position: 'relative' }}>
      <svg width={svgW} height={svgH} style={{ display: 'block' }}>
        {/* Edges */}
        {links.map((l, i) => {
          const s = positions[l.source], t = positions[l.target];
          if (!s || !t) return null;
          const edgeColors = { parent: '#7264e8', depends_on: '#e8a030', blocks: '#e85030', member_of: '#38c6e8' };
          const color = edgeColors[l.kind] || '#555';
          const mx = (s.x + t.x) / 2, my = (s.y + t.y) / 2;
          return (
            <g key={i}>
              <path d={`M${s.x},${s.y} Q${mx},${my - 30} ${t.x},${t.y}`}
                fill="none" stroke={color + '77'} strokeWidth="1.5"
                strokeDasharray={l.kind === 'depends_on' ? '5,3' : undefined} />
            </g>
          );
        })}
        {/* Nodes */}
        {artifacts.map(a => {
          const pos = positions[a.path];
          if (!pos) return null;
          const color = typeColors[a.type] || '#888';
          return (
            <g key={a.path} style={{ cursor: 'pointer' }} onClick={() => onNodeClick(a)}>
              <rect x={pos.x - 80} y={pos.y - 26} width={160} height={52} rx={8}
                fill={color + '22'} stroke={color} strokeWidth="1.5" />
              <text x={pos.x} y={pos.y - 8} textAnchor="middle"
                fill={color} fontSize="11" fontWeight="700" fontFamily="JetBrains Mono, monospace">
                {a.type}
              </text>
              <text x={pos.x} y={pos.y + 10} textAnchor="middle"
                fill="var(--text)" fontSize="10" fontFamily="Inter, sans-serif">
                {a.title.length > 20 ? a.title.slice(0, 19) + '…' : a.title}
              </text>
              <text x={pos.x} y={pos.y + 22} textAnchor="middle"
                fill="var(--text-faint)" fontSize="9" fontFamily="JetBrains Mono, monospace">
                {a.status}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}

// ── Artifact Modal (node click) ──
function ArtifactModal({ artifact, onClose, onEdit, onChangeState, onRunAgent, addToast }) {
  const [tab, setTab] = React.useState('preview');

  if (!artifact) return null;

  const transitions = {
    'draft':          ['clarifying', 'abandoned'],
    'clarifying':     ['planning', 'abandoned'],
    'planning':       ['in-development', 'rejected', 'abandoned'],
    'in-development': ['in-qa', 'rejected', 'abandoned'],
    'in-qa':          ['approved', 'rejected', 'abandoned'],
    'approved':       ['done', 'abandoned'],
    'rejected':       ['planning', 'abandoned'],
    'done':           [],
    'abandoned':      [],
  };
  const available = transitions[artifact.status] || [];

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 3000,
      background: 'oklch(0 0 0 / 0.65)', backdropFilter: 'blur(6px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: 24,
    }} onClick={onClose}>
      <div onClick={e => e.stopPropagation()} style={{
        background: 'var(--bg-elevated)', border: '1px solid var(--border)',
        borderRadius: 'var(--r-xl)', width: '100%', maxWidth: 780, maxHeight: '88vh',
        display: 'flex', flexDirection: 'column', boxShadow: 'var(--shadow)',
        overflow: 'hidden',
      }}>
        {/* Header */}
        <div style={{
          padding: '16px 20px', borderBottom: '1px solid var(--border-subtle)',
          display: 'flex', alignItems: 'center', gap: 10,
        }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
              <TypeBadge type={artifact.type} />
              <StatusBadge status={artifact.status} />
              {artifact.lineage && (
                <span style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)',
                  fontFamily: 'var(--font-mono)', display: 'flex', alignItems: 'center', gap: 4 }}>
                  <Icon name="git" size={11} /> {artifact.lineage}
                </span>
              )}
            </div>
            <div style={{ fontWeight: 700, fontSize: 'var(--text-xl)' }}>{artifact.title}</div>
            <div style={{ fontSize: 'var(--text-xs)', color: 'var(--text-faint)',
              fontFamily: 'var(--font-mono)', marginTop: 4 }}>
              lifecycle/{artifact.path}
            </div>
          </div>
          <button className="btn btn-ghost btn-icon" onClick={onClose}><Icon name="close" size={14} /></button>
        </div>

        {/* Action bar */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 6, padding: '10px 20px',
          borderBottom: '1px solid var(--border-subtle)', flexWrap: 'wrap',
        }}>
          <button className="btn btn-primary btn-sm" onClick={() => { onClose(); onEdit(artifact); }}>
            <Icon name="edit" size={12} /> Edit
          </button>
          {available.length > 0 && (
            <div style={{ position: 'relative', display: 'inline-flex', gap: 4 }}>
              {available.map(s => (
                <button key={s} className="btn btn-ghost btn-sm"
                  onClick={() => { onChangeState(artifact, s); addToast(`Status changed to ${s}`, 'success'); onClose(); }}>
                  → {s}
                </button>
              ))}
            </div>
          )}
          <button className="btn btn-ghost btn-sm" onClick={() => { onRunAgent(artifact); onClose(); }}>
            <Icon name="bot" size={12} /> Run Agent
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => addToast('Opening in editor…', 'info')}>
            <Icon name="terminal" size={12} /> Open in IDE
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => addToast('Git history: 4 commits', 'info')}>
            <Icon name="git" size={12} /> History
          </button>
          <button className="btn btn-danger btn-sm" onClick={() => addToast('Delete flow would open here', 'info')}>
            <Icon name="trash" size={12} /> Delete
          </button>
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid var(--border-subtle)', padding: '0 20px' }}>
          {['preview', 'frontmatter', 'links'].map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              background: 'none', border: 'none', cursor: 'pointer',
              padding: '10px 14px', fontSize: 'var(--text-sm)', fontWeight: tab === t ? 600 : 400,
              color: tab === t ? 'var(--accent)' : 'var(--text-muted)',
              borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
              marginBottom: -1, transition: 'all var(--dur-fast) var(--ease)',
              textTransform: 'capitalize',
            }}>{t}</button>
          ))}
        </div>

        {/* Tab content */}
        <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
          {tab === 'preview' && (
            <div style={{ lineHeight: 1.7, color: 'var(--text)', fontSize: 'var(--text-base)' }}>
              {artifact.body ? artifact.body.split('\n').map((line, i) => {
                if (line.startsWith('## ')) return <h2 key={i} style={{ fontSize: 'var(--text-lg)', fontWeight: 700, marginTop: i > 0 ? 20 : 0, marginBottom: 8, color: 'var(--text)' }}>{line.slice(3)}</h2>;
                if (line.startsWith('- ')) return <li key={i} style={{ marginLeft: 16, marginBottom: 4, color: 'var(--text-muted)' }}>{line.slice(2)}</li>;
                if (line.startsWith('`')) return <code key={i} style={{ fontFamily: 'var(--font-mono)', background: 'var(--bg-overlay)', padding: '0 4px', borderRadius: 3, fontSize: 12 }}>{line}</code>;
                return line ? <p key={i} style={{ marginBottom: 6 }}>{line}</p> : <br key={i} />;
              }) : <div style={{ color: 'var(--text-faint)' }}>No body content.</div>}
            </div>
          )}
          {tab === 'frontmatter' && (
            <div style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: '10px 16px', fontSize: 'var(--text-sm)' }}>
              {[
                ['title', artifact.title],
                ['type', artifact.type],
                ['status', artifact.status],
                ['lineage', artifact.lineage],
                ['release', artifact.release],
                ['sprint', artifact.sprint],
                ['labels', artifact.labels?.join(', ')],
                ['parent', artifact.parent],
                ['depends_on', artifact.depends_on?.join(', ')],
                ['assignees', artifact.assignees?.map(a => `${a.role}: ${a.who}`).join(', ')],
              ].filter(([, v]) => v).map(([k, v]) => (
                <React.Fragment key={k}>
                  <div style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-faint)', fontSize: 11, paddingTop: 1 }}>{k}:</div>
                  <div style={{ color: 'var(--text)', fontFamily: 'var(--font-mono)', fontSize: 12 }}>{v}</div>
                </React.Fragment>
              ))}
            </div>
          )}
          {tab === 'links' && (
            <div>
              <div style={{ fontSize: 'var(--text-xs)', fontWeight: 700, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 10 }}>Outbound</div>
              {[...(artifact.depends_on || []), ...(artifact.blocks || [])].length === 0
                ? <div style={{ color: 'var(--text-faint)', fontSize: 'var(--text-sm)' }}>No outbound links.</div>
                : [...(artifact.depends_on || []).map(p => ({ path: p, kind: 'depends_on' })),
                   ...(artifact.blocks || []).map(p => ({ path: p, kind: 'blocks' }))].map((l, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 0',
                    borderBottom: '1px solid var(--border-subtle)', fontSize: 'var(--text-sm)' }}>
                    <span style={{ padding: '1px 7px', borderRadius: 'var(--r-full)', fontSize: 10,
                      background: EDGE_COLORS[l.kind] + '33', color: EDGE_COLORS[l.kind], fontFamily: 'var(--font-mono)' }}>{l.kind}</span>
                    <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>{l.path}</span>
                  </div>
                ))
              }
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { GraphView, Graph2DView, ArtifactModal });
