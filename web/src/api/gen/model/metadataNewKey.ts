export interface MetadataNewKey {
  /**
   * @minLength 1
   * @maxLength 256
   */
  key: string;
  /**
   * A subscriber PIN to send beside the key, for a provider that takes one; left out, none
   * @maxLength 256
   */
  pin?: string;
}
