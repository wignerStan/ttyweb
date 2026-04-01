const STYLE: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  height: '100%',
  color: 'var(--text-muted)',
  fontSize: 14,
}

export function LazyFallback() {
  return <div style={STYLE}>Loading&hellip;</div>
}
