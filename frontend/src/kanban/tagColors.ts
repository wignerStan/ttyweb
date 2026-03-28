const TAG_COLORS = ['blue', 'purple', 'cyan', 'green', 'yellow', 'red'] as const;
type TagColor = (typeof TAG_COLORS)[number];

const TAG_COLOR_MAP: Record<string, TagColor> = {
  bug: 'red',
  feature: 'blue',
  enhancement: 'purple',
  docs: 'cyan',
  refactor: 'yellow',
  performance: 'green',
  security: 'red',
  design: 'purple',
  testing: 'cyan',
  devops: 'yellow',
  ux: 'blue',
  api: 'green',
};

export function getTagColorClass(tag: string): string {
  const lower = tag.toLowerCase().trim();
  const mapped = TAG_COLOR_MAP[lower];
  if (mapped) {
    return `tag--${mapped}`;
  }
  let hash = 0;
  for (let i = 0; i < lower.length; i++) {
    hash = lower.charCodeAt(i) + ((hash << 5) - hash);
  }
  return `tag--${TAG_COLORS[Math.abs(hash) % TAG_COLORS.length]}`;
}
