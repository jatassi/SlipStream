import type { QueryClient } from "@tanstack/react-query";

import { askPasskeyPrompt } from "../../account/passkey-prompt";
import type { IdentitySignedIn } from "../../api/gen/model";
import { pinErrors } from "../../session/pin-errors";
import { safeRedirect } from "../../session/safe-redirect";
import { markSignedIn } from "../../session/session";
import { PasskeyRefused } from "../../session/webauthn";
import { writePref } from "../../shell/prefs";
import type { SubmitErrors } from "../../ui/form-context";

export const signInErrors = (error: unknown): SubmitErrors =>
  error instanceof PasskeyRefused ? { form: error.message, fields: {} } : pinErrors(error);

/** Where a sign-in answer goes: Set a new PIN when the PIN is off, else where the person was headed. */
export function signedInTo(
  queryClient: QueryClient,
  session: IdentitySignedIn,
  redirect: string | undefined,
): string {
  writePref("lastEmail", session.account.email);
  markSignedIn(queryClient, session);
  if (session.passkeyPrompt) {
    askPasskeyPrompt();
  }
  return session.pinRequired ? "/new-pin" : safeRedirect(redirect);
}
