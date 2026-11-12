import { createBrowserRouter, type RouteObject } from 'react-router'
import Layout from './components/Layout'
import ConfirmSignUp from './pages/ConfirmSignUp'
import Home from './pages/Home'
import NotFound from './pages/NotFound'
import SignIn from './pages/SignIn'
import SignUp from './pages/SignUp'

// The route table, exported on its own so tests can render it in a memory router.
export const routes: RouteObject[] = [
  {
    element: <Layout />,
    children: [
      { path: '/', element: <Home /> },
      { path: '/signin', element: <SignIn /> },
      { path: '/signup', element: <SignUp /> },
      { path: '/confirm', element: <ConfirmSignUp /> },
      { path: '*', element: <NotFound /> }, // anything else
    ],
  },
]

export const router = createBrowserRouter(routes)