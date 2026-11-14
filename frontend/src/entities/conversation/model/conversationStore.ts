import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ChatMessage, Conversation } from './types'

interface ConversationState {
  conversations: Conversation[]
  messages: Record<string, ChatMessage[]>
  // Гидратация с backend: заменить список диалогов и/или сообщения.
  setConversations: (conversations: Conversation[]) => void
  upsertConversation: (conversation: Conversation) => void
  setMessages: (conversationId: string, messages: ChatMessage[]) => void
  createConversation: (title?: string) => string
  deleteConversation: (id: string) => void
  renameConversation: (id: string, title: string) => void
  appendMessage: (conversationId: string, message: ChatMessage) => void
  updateMessage: (
    conversationId: string,
    messageId: string,
    patch: Partial<ChatMessage>,
  ) => void
  getMessages: (conversationId: string) => ChatMessage[]
}

function newId(prefix: string) {
  return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}

function deriveTitle(text: string) {
  const clean = text.replace(/\s+/g, ' ').trim()
  return clean.length > 60 ? `${clean.slice(0, 57)}…` : clean
}

export const useConversationStore = create<ConversationState>()(
  persist(
    (set, get) => ({
      conversations: [],
      messages: {},

      setConversations: (conversations) => set({ conversations }),

      upsertConversation: (conversation) =>
        set((s) => {
          const exists = s.conversations.some((c) => c.id === conversation.id)
          return {
            conversations: exists
              ? s.conversations.map((c) => (c.id === conversation.id ? conversation : c))
              : [conversation, ...s.conversations],
          }
        }),

      setMessages: (conversationId, msgs) =>
        set((s) => ({ messages: { ...s.messages, [conversationId]: msgs } })),

      createConversation: (title) => {
        const id = newId('conv')
        const now = Date.now()
        set((s) => ({
          conversations: [
            {
              id,
              title: title ? deriveTitle(title) : 'Новый диалог',
              createdAt: now,
              updatedAt: now,
            },
            ...s.conversations,
          ],
        }))
        return id
      },

      deleteConversation: (id) =>
        set((s) => {
          const messages = { ...s.messages }
          delete messages[id]
          return {
            conversations: s.conversations.filter((c) => c.id !== id),
            messages,
          }
        }),

      renameConversation: (id, title) =>
        set((s) => ({
          conversations: s.conversations.map((c) =>
            c.id === id ? { ...c, title: deriveTitle(title), updatedAt: Date.now() } : c,
          ),
        })),

      appendMessage: (conversationId, message) =>
        set((s) => ({
          messages: {
            ...s.messages,
            [conversationId]: [...(s.messages[conversationId] ?? []), message],
          },
          conversations: s.conversations.map((c) =>
            c.id === conversationId ? { ...c, updatedAt: Date.now() } : c,
          ),
        })),

      updateMessage: (conversationId, messageId, patch) =>
        set((s) => ({
          messages: {
            ...s.messages,
            [conversationId]: (s.messages[conversationId] ?? []).map((m) =>
              m.id === messageId ? { ...m, ...patch } : m,
            ),
          },
        })),

      getMessages: (conversationId) => get().messages[conversationId] ?? [],
    }),
    { name: 'phystech.conversations.v1' },
  ),
)
