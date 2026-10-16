import type { OwnPatchInputBodyKeyPrompt } from "./ownPatchInputBodyKeyPrompt.ts";

export interface OwnPatchInputBody {
  /**
   * @minLength 1
   * @maxLength 100
   */
  displayName?: string;
  /** Whether the prompt to add a passkey may show after sign-in */
  keyPrompt?: OwnPatchInputBodyKeyPrompt;
}
