import { describe, expect, it } from 'vitest'
import { readConfig } from './config'

describe('readConfig', () => {
  it('returns the Cognito settings when both are set', () => {
    expect(
      readConfig({ VITE_COGNITO_USER_POOL_ID: 'ap-southeast-2_abc', VITE_COGNITO_CLIENT_ID: ' client123 ' }),
    ).toEqual({ userPoolId: 'ap-southeast-2_abc', userPoolClientId: 'client123' })
  })

  it('names every missing or blank variable', () => {
    expect(() => readConfig({ VITE_COGNITO_CLIENT_ID: '   ' })).toThrow(
      'Missing frontend config: VITE_COGNITO_USER_POOL_ID, VITE_COGNITO_CLIENT_ID',
    )
  })
})