import { Link, Outlet } from 'react-router'

// Shared frame for every page: header with nav, then the current page in <Outlet />.
export default function Layout() {
  return (
    <div className="min-h-screen bg-stone-50 text-stone-900">
      <header className="border-b border-stone-200 bg-white">
        <nav className="mx-auto flex max-w-3xl items-center gap-4 p-4">
          <Link to="/" className="font-semibold text-red-600">Inkwell</Link>
          <Link to="/signin" className="ml-auto text-sm">Sign in</Link>
          <Link to="/signup" className="text-sm">Sign up</Link>
        </nav>
      </header>
      <main className="mx-auto max-w-3xl p-4">
        <Outlet />
      </main>
    </div>
  )
}