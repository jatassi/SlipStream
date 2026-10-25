import { useQuery, useSuspenseQuery } from "@tanstack/react-query";

import { isProblem } from "../api/errors";
import type { IdentityPasskey, IdentitySignedIn, SystemPasskeys } from "../api/gen/model";
import {
  finishPasskeyReconfirm,
  finishPasskeyRegistration,
  finishPasskeySignIn,
  startPasskeyReconfirm,
  startPasskeyRegistration,
  startPasskeySignIn,
} from "../api/gen/query/identity/identity";
import { statusQuery } from "./session";
import { PasskeyRefused, webauthn } from "./webauthn";

/** Whether this address and this browser can hold a passkey, and in plain words why not. */
export type PasskeySupport =
  | { readonly available: true }
  | { readonly available: false; readonly why: string };

const whyNot: { readonly [R in NonNullable<SystemPasskeys["reason"]>]: string } = {
  ip_address:
    "Passkeys need SlipStream's name, not an IP address. Open SlipStream by its name to use one.",
  plain_http: "Passkeys need a secure address. Open SlipStream at its https:// address to use one.",
  not_a_domain:
    "Passkeys need a full name such as slipstream.example.com. Open SlipStream at one to use a passkey.",
};

const noBrowserSupport = "This browser can't use passkeys here.";

export function passkeySupport(passkeys: SystemPasskeys): PasskeySupport {
  if (!passkeys.available) {
    return {
      available: false,
      why: passkeys.reason === undefined ? noBrowserSupport : whyNot[passkeys.reason],
    };
  }
  return webauthn.supported() ? { available: true } : { available: false, why: noBrowserSupport };
}

export const usePasskeySupport = (): PasskeySupport =>
  passkeySupport(useSuspenseQuery(statusQuery).data.passkeys);

/** The support once the status has come back, for a control that mustn't hold up its page. */
export function usePasskeySupportSoon(): PasskeySupport | undefined {
  const { data } = useQuery(statusQuery);
  return data === undefined ? undefined : passkeySupport(data.passkeys);
}

/** Signs in with a discoverable passkey. Null means the person backed out or the call was aborted. */
export async function signInWithPasskey({
  mediation,
  signal,
}: {
  mediation?: CredentialMediationRequirement;
  signal?: AbortSignal;
} = {}): Promise<IdentitySignedIn | null> {
  const { ceremony, options } = await startPasskeySignIn(signal === undefined ? {} : { signal });
  const credential = await webauthn.get(options, {
    ...(mediation === undefined ? {} : { mediation }),
    ...(signal === undefined ? {} : { signal }),
  });
  return credential === null ? null : finishPasskeySignIn(ceremony, credential);
}

/** Registers a passkey for this address. Null means the person backed out. */
export async function addPasskey(): Promise<IdentityPasskey | null> {
  const { ceremony, options } = await startPasskeyRegistration();
  const credential = await webauthn.create(options);
  return credential === null ? null : finishPasskeyRegistration(ceremony, credential);
}

/**
 * Confirms it's the person with a passkey. False means they backed out, and a
 * PasskeyRefused says why the PIN is the way instead.
 */
export async function reconfirmWithPasskey(): Promise<boolean> {
  try {
    const { ceremony, options } = await startPasskeyReconfirm();
    const credential = await webauthn.get(options, {});
    if (credential === null) {
      return false;
    }
    await finishPasskeyReconfirm(ceremony, credential);
    return true;
  } catch (error) {
    if (isProblem(error, "passkey_unavailable")) {
      throw new PasskeyRefused("You have no passkey for this address. Enter your PIN.", {
        cause: error,
      });
    }
    throw error;
  }
}
