import React from 'react';

interface WorktreeStatusProps {
  ahead?: number;
  behind?: number;
  modified?: number;
  staged?: number;
  untracked?: number;
  conflicts?: number;
}

function isClean(props: WorktreeStatusProps): boolean {
  return (
    (props.ahead ?? 0) === 0 &&
    (props.behind ?? 0) === 0 &&
    (props.modified ?? 0) === 0 &&
    (props.staged ?? 0) === 0 &&
    (props.untracked ?? 0) === 0 &&
    (props.conflicts ?? 0) === 0
  );
}

function Badge({
  label,
  count,
  color,
}: {
  label: string;
  count: number;
  color: string;
}) {
  if (count === 0) return null;
  return (
    <span style={badgeStyle(color)}>
      {label} {count}
    </span>
  );
}

function badgeStyle(color: string): React.CSSProperties {
  return {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '2px',
    fontSize: '10px',
    lineHeight: '1',
    padding: '2px 6px',
    borderRadius: '3px',
    color,
    backgroundColor: `${color}18`,
    border: `1px solid ${color}40`,
    whiteSpace: 'nowrap' as const,
  };
}

export function WorktreeStatus(props: WorktreeStatusProps) {
  if (isClean(props)) {
    return (
      <span style={badgeStyle('#9ece6a')}>
        clean
      </span>
    );
  }

  return (
    <span style={containerStyle}>
      <Badge
        label="ahead"
        count={props.ahead ?? 0}
        color="#7aa2f7"
      />
      <Badge
        label="behind"
        count={props.behind ?? 0}
        color="#e0af68"
      />
      <Badge
        label="modified"
        count={props.modified ?? 0}
        color="#e0af68"
      />
      <Badge
        label="staged"
        count={props.staged ?? 0}
        color="#7dcfff"
      />
      <Badge
        label="untracked"
        count={props.untracked ?? 0}
        color="#bb9af7"
      />
      <Badge
        label="conflicts"
        count={props.conflicts ?? 0}
        color="#f7768e"
      />
    </span>
  );
}

const containerStyle: React.CSSProperties = {
  display: 'inline-flex',
  flexWrap: 'wrap',
  gap: '4px',
  alignItems: 'center',
};
