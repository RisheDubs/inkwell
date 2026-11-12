import { Amplify } from 'aws-amplify'
import type { AppConfig } from './config'

// Points Amplify Auth at our Cognito user pool. Call once, before the app renders.
export function configureAmplify(config: AppConfig): void {
  Amplify.configure({
    Auth: {
      Cognito: {
        userPoolId: config.userPoolId,
        userPoolClientId: config.userPoolClientId, // public client, no secret
        loginWith: { email: true },
        signUpVerificationMethod: 'code',
      },
    },
  })
}