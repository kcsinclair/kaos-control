// Parse an artifact date string robustly.
// Plain date strings like "2026-04-27" are interpreted as local midnight
// rather than UTC midnight (which is what new Date("YYYY-MM-DD") would do,
// producing an off-by-one-day error in timezones west of UTC).
// RFC3339 strings like "2026-04-27T00:00:00+10:00" carry explicit timezone
// info and are passed through to the Date constructor unchanged.
export function parseArtifactDate(iso: string | undefined): Date | null {
  if (!iso) return null
  // Plain date: YYYY-MM-DD with no time component
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
    const d = new Date(iso + 'T00:00:00')
    return isNaN(d.getTime()) ? null : d
  }
  const d = new Date(iso)
  return isNaN(d.getTime()) ? null : d
}

export function formatShortDate(iso: string | undefined): string {
  const d = parseArtifactDate(iso)
  if (!d) return '—'
  return d.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })
}

export function formatFullDateTime(iso: string | undefined): string {
  const d = parseArtifactDate(iso)
  if (!d) return '—'
  return d.toLocaleString(undefined, {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
    timeZoneName: 'short',
  })
}

export function useFormatDate() {
  return { formatShortDate, formatFullDateTime }
}
