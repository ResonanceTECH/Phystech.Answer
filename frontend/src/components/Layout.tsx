import { NavLink, Outlet } from 'react-router-dom'

export function Layout() {
  return (
    <div className="flex min-h-svh flex-col">
      <header className="border-b border-neutral-200 bg-white">
        <div className="mx-auto flex h-14 w-full max-w-5xl items-center justify-between px-4">
          <NavLink to="/" className="text-base font-semibold tracking-tight">
            Phystech.Answer
          </NavLink>
          <nav className="flex items-center gap-4 text-sm text-neutral-600">
            <NavLink
              to="/"
              className={({ isActive }) =>
                isActive ? 'text-neutral-900' : 'hover:text-neutral-900'
              }
            >
              Главная
            </NavLink>
          </nav>
        </div>
      </header>
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
