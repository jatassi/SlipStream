export interface FilesClientError {
  /** The external Download Client's name */
  client: string;
  /** When the Grab fails unless the error clears */
  failsAt: string;
  /** The error as the client words it */
  message: string;
  /** The client's Node's name */
  node: string;
}
