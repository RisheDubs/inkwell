// Types for import.meta.env. Optional, because a missing .env must be caught at runtime.
interface ImportMetaEnv {
  readonly VITE_COGNITO_USER_POOL_ID?: string
  readonly VITE_COGNITO_CLIENT_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}