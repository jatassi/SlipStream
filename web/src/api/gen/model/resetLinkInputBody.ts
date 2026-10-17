export interface ResetLinkInputBody {
  /** Also remove the Account's passkeys, for when someone else may have had access */
  removePasskeys?: boolean;
}
