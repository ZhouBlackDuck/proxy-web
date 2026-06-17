import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSettingsStore = defineStore('settings', () => {
  const theme = ref<'dark' | 'light'>(
    (localStorage.getItem('theme') as 'dark' | 'light') || 'dark'
  )
  const language = ref<string>(localStorage.getItem('language') || 'zh')

  function setTheme(t: 'dark' | 'light') {
    theme.value = t
    localStorage.setItem('theme', t)
  }

  function setLanguage(lang: string) {
    language.value = lang
    localStorage.setItem('language', lang)
  }

  return { theme, language, setTheme, setLanguage }
})
