import { useEffect, useEffectEvent, useRef, useState } from "react";

import type { IdentitySignedIn } from "../../api/gen/model";
import { signInWithPasskey, usePasskeySupport } from "../../session/passkey";
import { webauthn } from "../../session/webauthn";

type Outcome = {
  signedIn: (session: IdentitySignedIn) => Promise<void>;
  failed: (error: unknown) => void;
};

async function waitForAutofill({ signedIn, failed }: Outcome, signal: AbortSignal) {
  try {
    if (!(await webauthn.conditionalMediation())) {
      return;
    }
    const session = await signInWithPasskey({ mediation: "conditional", signal });
    if (session !== null) {
      await signedIn(session);
    }
  } catch (error) {
    if (!signal.aborted) {
      failed(error);
    }
  }
}

// The browser offers passkeys in the email field's autofill while a
// conditional ceremony waits. It runs one ceremony at a time, so the button
// aborts the waiting one, which waits again once the button's has settled.
export function usePasskeySignIn(outcome: Outcome) {
  const { available } = usePasskeySupport();
  const waiting = useRef<AbortController | null>(null);
  const [busy, setBusy] = useState(false);
  const wait = (on: boolean) => {
    const controller = new AbortController();
    waiting.current = controller;
    if (on) {
      void waitForAutofill(outcome, controller.signal);
    }
    return controller;
  };
  const waitOnLoad = useEffectEvent(wait);
  useEffect(() => {
    waitOnLoad(available);
    return () => {
      waiting.current?.abort();
    };
  }, [available]);
  const withButton = async () => {
    waiting.current?.abort();
    setBusy(true);
    try {
      const session = await signInWithPasskey();
      if (session !== null) {
        await outcome.signedIn(session);
        return;
      }
    } catch (error) {
      outcome.failed(error);
    } finally {
      setBusy(false);
    }
    wait(true);
  };
  return { shown: available, busy, withButton };
}
