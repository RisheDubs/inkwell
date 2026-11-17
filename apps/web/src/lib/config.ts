// Reads and validates the public frontend config. A pure function (env passed in) so it's easy
// to test and can't crash a test file just by being imported.
export type AppConfig = {
  userPoolId: string
  userPoolClientId: string
}

type Env = Partial<Record<'VITE_COGNITO_USER_POOL_ID' | 'VITE_COGNITO_CLIENT_ID', string>>

const REQUIRED = ['VITE_COGNITO_USER_POOL_ID', 'VITE_COGNITO_CLIENT_ID'] as const

export function readConfig(env: Env): AppConfig {
  const missing = REQUIRED.filter((key) => !env[key]?.trim())
  if (missing.length > 0) {
    throw new Error(`Missing frontend config: ${missing.join(', ')} (see apps/web/.env.example)`)
  }
  return {
    userPoolId: env.VITE_COGNITO_USER_POOL_ID!.trim(),
    userPoolClientId: env.VITE_COGNITO_CLIENT_ID!.trim(),
  }
}