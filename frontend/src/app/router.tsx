import { createBrowserRouter } from 'react-router-dom'
import { AdminLayout } from '@/app/layouts/AdminLayout'
import { AppLayout } from '@/app/layouts/AppLayout'
import { ChatLayout } from '@/app/layouts/ChatLayout'
import { paths } from '@/app/router/routePaths'
import { AdminDashboardPage } from '@/pages/admin/dashboard/AdminDashboardPage'
import { AdminDocumentsPage } from '@/pages/admin/documents/AdminDocumentsPage'
import { AdminStubPage } from '@/pages/admin/stub/AdminStubPage'
import { DocumentUploadPage } from '@/pages/admin/upload/DocumentUploadPage'
import { ForgotPasswordPage } from '@/pages/auth/forgot-password/ForgotPasswordPage'
import { LoginPage } from '@/pages/auth/login/LoginPage'
import { RegisterPage } from '@/pages/auth/register/RegisterPage'
import { ChatPage } from '@/pages/chat/ChatPage'
import { DocumentDetailsPage } from '@/pages/document-details/DocumentDetailsPage'
import { DocumentsPage } from '@/pages/documents/DocumentsPage'
import { HomePage } from '@/pages/home/HomePage'
import { InstructionsPage } from '@/pages/instructions/InstructionsPage'
import { LandingPage } from '@/pages/landing/LandingPage'
import { NotFoundPage } from '@/pages/not-found/NotFoundPage'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: paths.app.home, element: <HomePage /> },
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
    element: <AdminLayout />,
    children: [
      { index: true, element: <AdminDashboardPage /> },
      { path: 'documents', element: <AdminDocumentsPage /> },
      { path: 'documents/upload', element: <DocumentUploadPage /> },
      {
        path: 'processing',
        element: (
          <AdminStubPage
            title="Обработка документов"
            description="Очередь индексации и статус обработки появится после подключения фоновых задач на backend."
          />
        ),
      },
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
  { path: paths.auth.login, element: <LoginPage /> },
  { path: paths.auth.register, element: <RegisterPage /> },
  { path: paths.auth.forgotPassword, element: <ForgotPasswordPage /> },
  { path: paths.home, element: <LandingPage /> },
])
