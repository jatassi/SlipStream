export interface BackupsPassphraseState {
  /**
   * When the Admin last showed they saved the current passphrase; null until then
   * @nullable
   */
  savedAt: string | null;
}
