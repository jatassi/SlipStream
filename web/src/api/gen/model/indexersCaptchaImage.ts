export interface IndexersCaptchaImage {
  /** Send it back with the code the image shows */
  attempt: string;
  /** When the attempt closes. Load a new captcha after it. */
  expiresAt: string;
  /** The captcha as a data URL */
  image: string;
}
