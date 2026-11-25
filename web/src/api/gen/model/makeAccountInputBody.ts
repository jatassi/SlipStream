export interface MakeAccountInputBody {
  /**
   * Write-only. The Member's password for the server's app, set on the server and never stored
   * @minLength 1
   * @maxLength 1024
   */
  password: string;
  /**
   * The name the Member signs in to the server's app with
   * @minLength 1
   * @maxLength 100
   */
  username: string;
}
