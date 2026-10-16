export interface IdentityCeremony {
  /** The id the finish carries in its path */
  ceremony: string;
  /** PublicKeyCredentialCreationOptionsJSON for a registration, PublicKeyCredentialRequestOptionsJSON for an assertion: the browser's parseCreationOptionsFromJSON or parseRequestOptionsFromJSON takes it */
  options: unknown;
}
