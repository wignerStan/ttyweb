import { describe, expect, it, vi, beforeEach } from 'vitest'

describe('i18n configuration', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
  })

  it('exports a configured i18n instance', async () => {
    const i18next = (await import('./index')).default

    expect(i18next).toBeDefined()
    expect(i18next.isInitialized).toBe(true)
  })

  it('defaults to English language', async () => {
    const i18next = (await import('./index')).default

    expect(i18next.language).toBe('en')
  })

  it('has fallback language set to English', async () => {
    const i18next = (await import('./index')).default

    // i18next v26 may normalize fallbackLng to an array
    const fallback = i18next.options.fallbackLng
    const langs = Array.isArray(fallback) ? fallback : [fallback]
    expect(langs).toContain('en')
  })

  it('supports Chinese language', async () => {
    const i18next = (await import('./index')).default
    await i18next.changeLanguage('zh')

    expect(i18next.language).toBe('zh')
    expect(i18next.t('app.title')).toBe('ttyweb')
    expect(i18next.t('theme.dark')).toBe('深色')
  })

  it('persists language changes to localStorage', async () => {
    const i18next = (await import('./index')).default
    await i18next.changeLanguage('zh')

    expect(localStorage.getItem('ttyweb-lang')).toBe('zh')
  })

  it('restores language from localStorage', async () => {
    localStorage.setItem('ttyweb-lang', 'zh')
    vi.resetModules()
    const i18next = (await import('./index')).default

    expect(i18next.language).toBe('zh')
  })

  it('contains expected English keys', async () => {
    const i18next = (await import('./index')).default

    expect(i18next.t('app.title')).toBe('ttyweb')
    expect(i18next.t('sidebar.title')).toBe('Sessions')
    expect(i18next.t('terminal.title')).toBe('Terminal')
    expect(i18next.t('conversations.title')).toBe('Conversations')
    expect(i18next.t('notepad.title')).toBe('Notepad')
    expect(i18next.t('kanban.title')).toBe('Kanban')
    expect(i18next.t('theme.toggle')).toBe('Toggle theme')
    expect(i18next.t('mobile.paste')).toBe('Paste')
    expect(i18next.t('auth.login')).toBe('Login')
    expect(i18next.t('toolbar.esc')).toBe('Esc')
    expect(i18next.t('actions.save')).toBe('Save')
  })

  it('contains expected Chinese keys', async () => {
    const i18next = (await import('./index')).default
    await i18next.changeLanguage('zh')

    expect(i18next.t('app.title')).toBe('ttyweb')
    expect(i18next.t('sidebar.title')).toBe('会话')
    expect(i18next.t('terminal.title')).toBe('终端')
    expect(i18next.t('conversations.title')).toBe('对话')
    expect(i18next.t('notepad.title')).toBe('记事本')
    expect(i18next.t('kanban.title')).toBe('看板')
    expect(i18next.t('theme.toggle')).toBe('切换主题')
    expect(i18next.t('mobile.paste')).toBe('粘贴')
    expect(i18next.t('auth.login')).toBe('登录')
    expect(i18next.t('toolbar.esc')).toBe('Esc')
    expect(i18next.t('actions.save')).toBe('保存')
  })
})
