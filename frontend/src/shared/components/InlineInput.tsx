import { Check, X } from 'lucide-react'
import { forwardRef } from 'react'

interface InlineInputProps {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  onCancel: () => void
  placeholder?: string
  loading?: boolean
  name?: string
  /** Additional class for the input element */
  inputClassName?: string
  /** Additional class for the confirm button */
  confirmClassName?: string
  /** Additional class for the cancel button */
  cancelClassName?: string
  /** Additional class for the wrapper div */
  className?: string
  /** Icon size in pixels (default 14) */
  iconSize?: number
  /** data-testid for the confirm button */
  confirmTestId?: string
  /** data-testid for the cancel button */
  cancelTestId?: string
}

export const InlineInput = forwardRef<HTMLInputElement, InlineInputProps>(function InlineInput(
  {
    value,
    onChange,
    onSubmit,
    onCancel,
    placeholder,
    loading = false,
    name,
    inputClassName,
    confirmClassName,
    cancelClassName,
    className,
    iconSize = 14,
    confirmTestId,
    cancelTestId,
  },
  ref,
) {
  return (
    <div className={className ?? 'flex gap-1.5'}>
      <input
        ref={ref}
        className={inputClassName ?? 'input input-bordered input-sm flex-1 text-sm'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') onSubmit()
          if (e.key === 'Escape') onCancel()
        }}
        placeholder={placeholder}
        disabled={loading}
        name={name}
        autoComplete="off"
        type="text"
      />
      <button
        type="button"
        className={confirmClassName ?? 'btn btn-primary btn-sm btn-square'}
        onClick={onSubmit}
        disabled={loading || !value.trim()}
        aria-label="Confirm"
        data-testid={confirmTestId}
      >
        <Check size={iconSize} />
      </button>
      <button
        type="button"
        className={cancelClassName ?? 'btn btn-ghost btn-sm btn-square'}
        onClick={onCancel}
        aria-label="Cancel"
        data-testid={cancelTestId}
      >
        <X size={iconSize} />
      </button>
    </div>
  )
})
