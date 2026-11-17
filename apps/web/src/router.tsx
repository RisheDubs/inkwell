import type { ComponentType } from 'react'
import { createBrowserRouter, type RouteObject } from 'react-router'
import JournalFrame from './components/JournalFrame'
import Layout from './components/Layout'
import PublicFrame from './components/PublicFrame'
import RequireAuth from './components/RequireAuth'
import ConfirmSignUp from './pages/ConfirmSignUp'
import Home from './pages/Home'
import NotFound from './pages/NotFound'
import PublicJournal from './pages/PublicJournal'
import ResetPassword from './pages/ResetPassword'
import Settings from './pages/Settings'
import SignIn from './pages/SignIn'
import SignUp from './pages/SignUp'

// Pages that bring the Tiptap editor (most of the bundle) load on first visit, in their own
// chunk. So the sign-in page and a stranger's /j/{handle} list don't download an editor.
// Never import these pages statically elsewhere, or the bundler puts them back in the main chunk.
const page = (load: () => Promise<{ default: ComponentType }>) => async () => ({ Component: (await load()).default })
const EntryPage = page(() => import('./pages/EntryPage'))
const PublicEntry = page(() => import('./pages/PublicEntry'))

// The route table, exported on its own so tests can render it in a memory router.
export const routes: RouteObject[] = [
  {
    element: <Layout />,
    // Shown while a lazy page's chunk downloads on a first visit straight to it (e.g. a shared entry link).
    hydrateFallbackElement: (
      <p role="status" className="p-4 text-sm text-stone-600">
        Loading…
      </p>
    ),
    children: [
      // Signed-in only: the guard first, then the journal's theme around every page.
      {
        element: <RequireAuth />,
        children: [
          {
            element: <JournalFrame />,
            children: [
              { path: '/', element: <Home /> },
              { path: '/entries/new', lazy: EntryPage },
              { path: '/entries/:id', lazy: EntryPage },
              { path: '/settings', element: <Settings /> },
            ],
          },
        ],
      },
      // Shared journals: no sign-in, no token, public image route (ADR-0009).
      {
        path: '/j/:handle',
        element: <PublicFrame />,
        children: [
          { index: true, element: <PublicJournal /> },
          { path: ':id', lazy: PublicEntry },
        ],
      },
      // Public pages.
      { path: '/signin', element: <SignIn /> },
      { path: '/signup', element: <SignUp /> },
      { path: '/confirm', element: <ConfirmSignUp /> },
      { path: '/reset-password', element: <ResetPassword /> },
      { path: '*', element: <NotFound /> }, // anything else
    ],
  },
]

export const router = createBrowserRouter(routes)