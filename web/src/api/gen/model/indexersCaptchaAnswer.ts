export interface IndexersCaptchaAnswer {
  /**
   * The code the captcha shows
   * @maxLength 64
   */
  answer: string;
  /**
   * The attempt the captcha was loaded with
   * @minLength 1
   * @maxLength 64
   */
  attempt: string;
}
