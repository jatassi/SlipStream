export interface SendToInputBody {
  /** A downloading Node, for its built-in client; send this or clientId */
  builtInOn?: string;
  /** An external client; send this or builtInOn */
  clientId?: string;
}
