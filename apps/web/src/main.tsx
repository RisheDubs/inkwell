import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { readConfig } from './lib/config'
import { configureAmplify } from './lib/amplify'

// Fail fast: a missing .env value throws here with a clear message, before any page renders.
configureAmplify(readConfig(import.meta.env))

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)