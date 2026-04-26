// ── kc-common.jsx — shared UI primitives ──

function TypeBadge({ type }) {
  const color = TYPE_COLORS[type] || 'var(--text-faint)';
  const label = type || 'unknown';
  return (
    <span style={{
      display: 'inline-block', padding: '2px 7px', borderRadius: 'var(--r-full)',
      fontSize: '10px', fontWeight: 700, letterSpacing: '0.04em',
      fontFamily: 'var(--font-mono)', textTransform: 'uppercase',
      background: color, color: 'oklch(0.08 0 0)', whiteSpace: 'nowrap',
    }}>{label}</span>
  );
}

function StatusBadge({ status }) {
  const color = STATUS_COLORS[status] || 'var(--text-faint)';
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      padding: '2px 8px', borderRadius: 'var(--r-full)',
      fontSize: 'var(--text-xs)', fontWeight: 600, letterSpacing: '0.02em',
      fontFamily: 'var(--font-mono)', color,
      border: `1px solid ${color}`, background: `color-mix(in oklch, ${color} 12%, transparent)`,
      whiteSpace: 'nowrap',
    }}>
      <span style={{ width: 6, height: 6, borderRadius: '50%', background: color, flexShrink: 0 }}></span>
      {status}
    </span>
  );
}

function LabelChip({ label }) {
  return (
    <span style={{
      display: 'inline-block', padding: '1px 8px', borderRadius: 'var(--r-full)',
      fontSize: 'var(--text-xs)', fontWeight: 500, fontFamily: 'var(--font-mono)',
      background: 'var(--bg-overlay)', color: 'var(--text-muted)',
      border: '1px solid var(--border-subtle)',
    }}>{label}</span>
  );
}

function Avatar({ email, size = 24 }) {
  const initials = email ? email.slice(0, 2).toUpperCase() : '??';
  const hue = email ? (email.charCodeAt(0) * 37 + email.charCodeAt(1) * 17) % 360 : 180;
  return (
    <span title={email} style={{
      display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
      width: size, height: size, borderRadius: '50%', flexShrink: 0,
      background: `oklch(0.55 0.18 ${hue})`, color: 'white',
      fontSize: size * 0.38, fontWeight: 700, fontFamily: 'var(--font-sans)',
    }}>{initials}</span>
  );
}

function Icon({ name, size = 14 }) {
  const icons = {
    graph:    'M12 2a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm8 8a2 2 0 1 1 0 4 2 2 0 0 1 0-4zM4 10a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm8 8a2 2 0 1 1 0 4 2 2 0 0 1 0-4zM13.7 5.3l4.6 4.5M10.3 5.3 5.7 9.8M13.7 18.7l4.6-4.5M10.3 18.7l-4.6-4.5',
    list:     'M3 6h18M3 12h18M3 18h18',
    edit:     'M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z',
    bot:      'M12 2a3 3 0 0 0-3 3v1H6a3 3 0 0 0-3 3v8a3 3 0 0 0 3 3h12a3 3 0 0 0 3-3V9a3 3 0 0 0-3-3h-3V5a3 3 0 0 0-3-3zM9 12a1 1 0 1 1 2 0 1 1 0 0 1-2 0zm4 0a1 1 0 1 1 2 0 1 1 0 0 1-2 0z',
    settings: 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z',
    git:      'M18 3a3 3 0 0 0-2.82 4H8.82a3 3 0 1 0 0 2H12v3.18A3 3 0 1 0 14 15.82V9h1.18A3 3 0 1 0 18 3z',
    terminal: 'M4 17l6-6-6-6M12 19h8',
    trash:    'M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6',
    close:    'M18 6L6 18M6 6l12 12',
    check:    'M20 6L9 17l-5-5',
    arrow:    'M5 12h14M12 5l7 7-7 7',
    plus:     'M12 5v14M5 12h14',
    warning:  'M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0zM12 9v4M12 17h.01',
    info:     'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 8h.01M11 12h1v4h1',
    filter:   'M22 3H2l8 9.46V19l4 2v-8.54L22 3z',
    moon:     'M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z',
    sun:      'M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42M12 5a7 7 0 1 0 0 14A7 7 0 0 0 12 5z',
    play:     'M5 3l14 9-14 9V3z',
    stop:     'M6 6h12v12H6z',
    zap:      'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
    eye:      'M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8zM12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
    folder:   'M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z',
    chevron:  'M9 18l6-6-6-6',
    lock:     'M19 11H5a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7a2 2 0 0 0-2-2zM7 11V7a5 5 0 0 1 10 0v4',
    refresh:  'M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15',
    copy:     'M8 4H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-4-4h-6zM14 2v6h6M9 13h6M9 17h4',
    kill:     'M18 6L6 18M6 6l12 12',
    merge:    'M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5',
  };
  const d = icons[name] || icons.info;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none"
      stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"
      style={{ flexShrink: 0 }}>
      <path d={d} />
    </svg>
  );
}

function Spinner({ size = 16 }) {
  return (
    <span style={{
      display: 'inline-block', width: size, height: size,
      border: `2px solid var(--border)`,
      borderTopColor: 'var(--accent)',
      borderRadius: '50%',
      animation: 'kc-spin 0.7s linear infinite',
      flexShrink: 0,
    }} />
  );
}

function Toast({ toasts, dismiss }) {
  return (
    <div style={{
      position: 'fixed', bottom: 24, right: 24, zIndex: 9999,
      display: 'flex', flexDirection: 'column', gap: 8, pointerEvents: 'none',
    }}>
      {toasts.map(t => (
        <div key={t.id} style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '10px 16px', borderRadius: 'var(--r-lg)',
          background: 'var(--bg-overlay)', backdropFilter: 'blur(12px)',
          border: '1px solid var(--border)', boxShadow: 'var(--shadow)',
          maxWidth: 360, pointerEvents: 'all',
          color: t.kind === 'error' ? 'oklch(0.70 0.22 22)' : t.kind === 'success' ? 'var(--status-done)' : 'var(--text)',
          animation: 'kc-slide-in 0.2s var(--ease)',
        }}>
          <Icon name={t.kind === 'error' ? 'warning' : t.kind === 'success' ? 'check' : 'info'} size={14} />
          <span style={{ fontSize: 'var(--text-sm)', flex: 1 }}>{t.message}</span>
          <button onClick={() => dismiss(t.id)} style={{
            background: 'none', border: 'none', cursor: 'pointer',
            color: 'var(--text-muted)', padding: 2, lineHeight: 1,
          }}><Icon name="close" size={12} /></button>
        </div>
      ))}
    </div>
  );
}

function useToasts() {
  const [toasts, setToasts] = React.useState([]);
  const add = React.useCallback((message, kind = 'info') => {
    const id = Date.now();
    setToasts(ts => [...ts, { id, message, kind }]);
    setTimeout(() => setToasts(ts => ts.filter(t => t.id !== id)), 4000);
  }, []);
  const dismiss = React.useCallback(id => setToasts(ts => ts.filter(t => t.id !== id)), []);
  return { toasts, add, dismiss };
}

function ConfirmDialog({ title, message, confirmLabel = 'Confirm', danger = false, onConfirm, onCancel }) {
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 8000,
      background: 'oklch(0 0 0 / 0.6)', backdropFilter: 'blur(4px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={onCancel}>
      <div onClick={e => e.stopPropagation()} style={{
        background: 'var(--bg-elevated)', border: '1px solid var(--border)',
        borderRadius: 'var(--r-xl)', padding: '24px', width: 380,
        boxShadow: 'var(--shadow)',
      }}>
        <div style={{ fontWeight: 600, marginBottom: 8 }}>{title}</div>
        <div style={{ color: 'var(--text-muted)', fontSize: 'var(--text-sm)', marginBottom: 20 }}>{message}</div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button className="btn btn-ghost btn-sm" onClick={onCancel}>Cancel</button>
          <button className={`btn btn-sm ${danger ? 'btn-danger' : 'btn-primary'}`} onClick={onConfirm}>{confirmLabel}</button>
        </div>
      </div>
    </div>
  );
}

function EmptyState({ icon = 'info', title, subtitle }) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      gap: 12, padding: 48, color: 'var(--text-faint)', textAlign: 'center',
    }}>
      <Icon name={icon} size={32} />
      <div>
        <div style={{ fontWeight: 600, color: 'var(--text-muted)', marginBottom: 4 }}>{title}</div>
        {subtitle && <div style={{ fontSize: 'var(--text-sm)' }}>{subtitle}</div>}
      </div>
    </div>
  );
}

function ProgressBar({ pct, color = 'var(--accent)' }) {
  return (
    <div style={{ height: 4, borderRadius: 'var(--r-full)', background: 'var(--bg-overlay)', overflow: 'hidden' }}>
      <div style={{
        height: '100%', width: `${pct}%`, background: color,
        borderRadius: 'var(--r-full)', transition: 'width 0.5s var(--ease)',
      }} />
    </div>
  );
}

function WsStatusDot({ status }) {
  const colors = { connected: 'var(--status-done)', connecting: 'var(--status-clarifying)', disconnected: 'oklch(0.60 0.22 22)' };
  return (
    <span title={`WebSocket: ${status}`} style={{
      width: 8, height: 8, borderRadius: '50%', display: 'inline-block',
      background: colors[status] || colors.disconnected,
      boxShadow: status === 'connected' ? '0 0 6px var(--status-done)' : 'none',
      animation: status === 'connecting' ? 'kc-pulse 1s ease infinite' : 'none',
    }} />
  );
}

Object.assign(window, {
  TypeBadge, StatusBadge, LabelChip, Avatar, Icon, Spinner,
  Toast, useToasts, ConfirmDialog, EmptyState, ProgressBar, WsStatusDot,
});
