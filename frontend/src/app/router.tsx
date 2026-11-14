import { createBrowserRouter } from 'react-router-dom'
import { AdminLayout } from '@/app/layouts/AdminLayout'
import { AppLayout } from '@/app/layouts/AppLayout'
import { ChatLayout } from '@/app/layouts/ChatLayout'
import { RequireAdmin, RequireAuth } from '@/app/router/guards'
import { paths } from '@/app/router/routePaths'
import { AdminDashboardPage } from '@/pages/admin/dashboard/AdminDashboardPage'
import { AdminDocumentsPage } from '@/pages/admin/documents/AdminDocumentsPage'
import { AdminStubPage } from '@/pages/admin/stub/AdminStubPage'
import { DocumentUploadPage } from '@/pages/admin/upload/DocumentUploadPage'
import { ChatPage } from '@/pages/chat/ChatPage'
import { DocumentDetailsPage } from '@/pages/document-details/DocumentDetailsPage'
import { DocumentsPage } from '@/pages/documents/DocumentsPage'
import { HomePage } from '@/pages/home/HomePage'
import { InstructionsPage } from '@/pages/instructions/InstructionsPage'
import { LandingPage } from '@/pages/landing/LandingPage'
import { LoginPage } from '@/pages/login/LoginPage'
import { OnboardingPage } from '@/pages/onboarding/OnboardingPage'
import { NotFoundPage } from '@/pages/not-found/NotFoundPage'

export const router = createBrowserRouter([
  {
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { path: paths.app.home, element: <HomePage /> },
      { path: paths.app.onboarding, element: <OnboardingPage /> },
      { path: paths.app.documents, element: <DocumentsPage /> },
      { path: paths.app.document(':documentId'), element: <DocumentDetailsPage /> },
      { path: paths.app.instructions, element: <InstructionsPage /> },
      {
        path: paths.app.chatNew,
        element: <ChatLayout />,
        children: [{ index: true, element: <ChatPage /> }],
      },
      {
        path: '/app/chat/:conversationId',
        element: <ChatLayout />,
        children: [{ index: true, element: <ChatPage /> }],
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
  {
    path: paths.admin.root,
    element: (
      <RequireAdmin>
        <AdminLayout />
      </RequireAdmin>
    ),
    children: [
      { index: true, element: <AdminDashboardPage /> },
      { path: 'documents', element: <AdminDocumentsPage /> },
      { path: 'documents/upload', element: <DocumentUploadPage /> },
      { path: 'processing', element: <DocumentUploadPage /> },
      {
        path: 'versions',
        element: (
          <AdminStubPage
            title="Версии документов"
            description="История редакций и сравнение версий — раздел в разработке, API на backend пока нет."
          />
        ),
      },
      {
        path: 'conflicts',
        element: (
          <AdminStubPage
            title="Конфликты документов"
            description="Выявление противоречий между регламентами — раздел в разработке, API на backend пока нет."
          />
        ),
      },
      {
        path: 'analytics',
        element: (
          <AdminStubPage
            title="Аналитика"
            description="Статистика запросов и популярных тем — раздел в разработке, API на backend пока нет."
          />
        ),
      },
    ],
  },
  { path: paths.home, element: <LandingPage /> },
  { path: paths.login, element: <LoginPage /> },
])
