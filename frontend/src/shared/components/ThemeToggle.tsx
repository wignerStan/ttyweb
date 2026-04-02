import { useTheme, THEMES } from '../../hooks/useTheme'

function ThemeIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M12 2v2" />
      <path d="M12 20v2" />
      <path d="m4.93 4.93 1.41 1.41" />
      <path d="m17.66 17.66 1.41 1.41" />
      <path d="M2 12h2" />
      <path d="M20 12h2" />
      <path d="m6.34 17.66-1.41 1.41" />
      <path d="m19.07 4.93-1.41 1.41" />
      <circle cx="12" cy="12" r="4" />
    </svg>
  )
}

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()

  return (
    <div className="dropdown dropdown-end dropdown-hover ml-auto">
      <div
        tabIndex={0}
        role="button"
        className="btn btn-ghost btn-xs btn-square text-base-content/75 hover:text-base-content"
        aria-label="Theme menu"
      >
        <ThemeIcon />
      </div>
      <ul
        tabIndex={0}
        className="dropdown-content z-[100] menu p-2 shadow bg-base-200 rounded-box w-36 mt-1 border border-base-300"
      >
        {THEMES.map((t) => (
          <li key={t.id}>
            <button
              type="button"
              className={theme === t.id ? 'active' : ''}
              onClick={() => setTheme(t.id)}
            >
              {t.label}
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
