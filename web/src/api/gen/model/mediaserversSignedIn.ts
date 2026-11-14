export interface MediaserversSignedIn {
  /** The key named SlipStream the sign-in created; send it as the Draft's apiKey */
  apiKey: string;
  /** The DeviceId the sign-in used; send it as the Draft's deviceId */
  deviceId: string;
}
