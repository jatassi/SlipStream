/**
 * The browser's half of a passkey ceremony. The Node's options are the W3C
 * JSON forms, and what goes back is the credential's toJSON(). Each call
 * answers null when the person cancelled or the call was aborted. Tests
 * replace these functions, since a browser without an authenticator can't
 * run them.
 */
export const webauthn = {
  supported: (): boolean => isSecureContext && "PublicKeyCredential" in globalThis,

  conditionalMediation: (): Promise<boolean> =>
    PublicKeyCredential.isConditionalMediationAvailable(),

  create: (options: unknown): Promise<unknown> =>
    settle(
      navigator.credentials.create({
        publicKey: PublicKeyCredential.parseCreationOptionsFromJSON(
          // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the Node answers PublicKeyCredentialCreationOptionsJSON
          options as PublicKeyCredentialCreationOptionsJSON,
        ),
      }),
    ),

  get: (
    options: unknown,
    { mediation, signal }: { mediation?: CredentialMediationRequirement; signal?: AbortSignal },
  ): Promise<unknown> =>
    settle(
      navigator.credentials.get({
        publicKey: PublicKeyCredential.parseRequestOptionsFromJSON(
          // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the Node answers PublicKeyCredentialRequestOptionsJSON
          options as PublicKeyCredentialRequestOptionsJSON,
        ),
        ...(mediation === undefined ? {} : { mediation }),
        ...(signal === undefined ? {} : { signal }),
      }),
    ),
};

/** The browser refused the ceremony for a reason other than the person backing out. */
export class PasskeyRefused extends Error {
  override readonly name = "PasskeyRefused";
}

const backedOut = new Set(["NotAllowedError", "AbortError"]);

async function settle(ceremony: Promise<Credential | null>): Promise<unknown> {
  try {
    const credential = await ceremony;
    return credential instanceof PublicKeyCredential ? credential.toJSON() : null;
  } catch (error) {
    if (error instanceof DOMException && backedOut.has(error.name)) {
      return null;
    }
    if (error instanceof DOMException && error.name === "InvalidStateError") {
      throw new PasskeyRefused("This device already has a passkey for you here.", { cause: error });
    }
    throw new PasskeyRefused("This browser couldn't use a passkey.", { cause: error });
  }
}
