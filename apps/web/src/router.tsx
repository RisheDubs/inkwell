import { createBrowserRouter, type RouteObject } from 'react-router'
import JournalFrame from './components/JournalFrame'
import Layout from './components/Layout'
import RequireAuth from './components/RequireAuth'
import ConfirmSignUp from './pages/ConfirmSignUp'
import EntryPage from './pages/EntryPage'
import Home from './pages/Home'
import NotFound from './pages/NotFound'
import ResetPassword from './pages/ResetPassword'
import Settings from './pages/Settings'
import SignIn from './pages/SignIn'
import SignUp from './pages/SignUp'

// The route table, exported on its own so tests can render it in a memory router.
export const routes: RouteObject[] = [
  {
    element: <Layout />,
    children: [
      // Signed-in only: the guard first, then the journal's theme around every page.
      {
        element: <RequireAuth />,
        children: [
          {
            element: <JournalFrame />,
            children: [
              { path: '/', element: <Home /> },
              { path: '/entries/new', element: <EntryPage /> },
              { path: '/entries/:id', element: <EntryPage /> },
              { path: '/settings', element: <Settings /> },
            ],
          },
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