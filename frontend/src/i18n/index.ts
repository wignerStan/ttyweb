import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './en.json'
import zh from './zh.json'

const STORAGE_KEY = 'ttyweb-lang'

function getSavedLanguage(): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved && (saved === 'en' || saved === 'zh')) {
      return saved
    }
  } catch {
    // localStorage unavailable
  }
  return 'en'
}

i18next.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    zh: { translation: zh },
  },
  lng: getSavedLanguage(),
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
})

i18next.on('languageChanged', (lng: string) => {
  try {
    localStorage.setItem(STORAGE_KEY, lng)
  } catch {
    // localStorage unavailable
  }
})

export default i18next
