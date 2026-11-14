export interface MediaserversSignInBody {
  /**
   * @minLength 1
   * @maxLength 2048
   */
  address: string;
  /** @minLength 1 */
  kind: string;
  /** The reaching Node the draft names. The Controller signs in when it reaches the server, else this Node does */
  nodeId: string;
  /**
   * Write-only. Used for this one sign-in and never stored
   * @maxLength 1024
   */
  password: string;
  /**
   * An administrator of the server
   * @minLength 1
   * @maxLength 256
   */
  user: string;
}
