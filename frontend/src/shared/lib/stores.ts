import { create } from 'zustand'

// Клиентское UI-состояние чата: выбранный источник и панель источников.
// Не дублирует серверные данные — только «что сейчас открыто на экране».
interface ChatUiState {
  sourcesOpen: boolean
  selectedDocumentId: string | null
  selectedQuote: string | null
  openSources: (documentId: string, quote?: string) => void
  closeSources: () => void
}

export const useChatUiStore = create<ChatUiState>()((set) => ({
  sourcesOpen: false,
  selectedDocumentId: null,
  selectedQuote: null,
  openSources: (documentId, quote) =>
    set({ sourcesOpen: true, selectedDocumentId: documentId, selectedQuote: quote ?? null }),
  closeSources: () =>
    set({ sourcesOpen: false, selectedDocumentId: null, selectedQuote: null }),
}))

// Глобальное UI-состояние: мобильная навигация.
interface AppUiState {
  mobileNavOpen: boolean
  setMobileNavOpen: (open: boolean) => void
}

export const useAppUiStore = create<AppUiState>()((set) => ({
  mobileNavOpen: false,
  setMobileNavOpen: (open) => set({ mobileNavOpen: open }),
}))
