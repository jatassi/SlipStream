export interface IndexersPastedCookie {
  /**
   * The Cookie header's value from a signed-in browser: name=value pairs separated by semicolons
   * @minLength 1
   * @maxLength 8192
   */
  cookie: string;
  /**
   * The User-Agent of that browser, sent with the cookie
   * @maxLength 512
   */
  userAgent?: string;
}
