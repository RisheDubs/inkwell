import { Amplify } from 'aws-amplify'
import { describe, expect, it } from 'vitest'
import { configureAmplify } from './amplify'

describe('configureAmplify', () => {
  it('points Auth at the given pool and client, signing in with email', () => {
    configureAmplify({ userPoolId: 'ap-southeast-2_abc', userPoolClientId: 'client123' })

    expect(Amplify.getConfig().Auth?.Cognito).toMatchObject({
      userPoolId: 'ap-southeast-2_abc',
      userPoolClientId: 'client123',
      loginWith: { email: true },
    })
  })
})