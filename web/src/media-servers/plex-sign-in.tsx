import { useQueryClient } from "@tanstack/react-query";
import { Suspense, useCallback, useState } from "react";

import { isApiError, sentenceOf } from "../api/errors";
import type {
  MediaserversPlexPin as PlexPin,
  MediaserversPlexPinStateState as PinState,
} from "../api/gen/model";
import { writePref } from "../shell/prefs";
import { Button } from "../ui/button";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";
import { replacePlexSignIn, startPlexTab, useWatchPin } from "./use-plex-pin";

/** Why the Admin signs in again: the last PIN expired, or plex.tv refused the sign-in SlipStream had. */
export type Lapse = "expired" | "refused";

const lapseWords: { readonly [L in Lapse]: string } = {
  expired: "The Plex sign-in expired before it was approved. Sign in with Plex again.",
  refused: "SlipStream's sign-in to Plex has expired or was revoked. Sign in with Plex again.",
};

/** Opens plex.tv in a new tab, where the Admin signs in. SlipStream never sees the password. */
export function SignInPart({
  lapse,
  onBack,
  onStarted,
}: {
  lapse: Lapse | undefined;
  onBack: () => void;
  onStarted: (pin: PlexPin) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const start = async () => {
    setBusy(true);
    setError(undefined);
    try {
      onStarted(await startPlexTab());
    } catch (failure: unknown) {
      setError(sentenceOf(failure));
      setBusy(false);
    }
  };
  return (
    <div className="flex flex-col gap-4">
      {lapse !== undefined && <SignalLine tone="warn">{lapseWords[lapse]}</SignalLine>}
      <p className="text-pretty text-ink-1">
        Plex opens in a new tab, where you sign in to your Plex account. Your password stays with
        Plex, and SlipStream lists your Plex servers here once you approve the sign-in.
      </p>
      {error !== undefined && <SignalLine tone="fail">{error}</SignalLine>}
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onBack}>Back</Button>
        <Button look="primary" busy={busy} onClick={() => void start()}>
          Sign in with Plex
        </Button>
      </div>
    </div>
  );
}

/** How a Plex sign-in ends for the step: signed in, or expired before the Admin approved it. */
export type PinEnd = "signed_in" | "expired";

function WatchPin({
  pinId,
  onSettled,
}: {
  pinId: number;
  onSettled: (state: Exclude<PinState, "waiting">) => void;
}) {
  useWatchPin(pinId, onSettled);
  return null;
}

/** The sign-in named a Plex account other than SlipStream's: replacing it removes the Plex servers. */
function OtherAccountPart({
  pin,
  onBack,
  onSettled,
}: {
  pin: PlexPin;
  onBack: () => void;
  onSettled: (state: PinEnd) => void;
}) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const replace = async () => {
    setBusy(true);
    setError(undefined);
    try {
      await replacePlexSignIn(queryClient, pin.pinId);
      onSettled("signed_in");
    } catch (failure: unknown) {
      if (isApiError(failure) && failure.code === "plex_pin_not_found") {
        writePref("plexPin", null);
        onSettled("expired");
        return;
      }
      setError(sentenceOf(failure));
      setBusy(false);
    }
  };
  return (
    <div className="flex flex-col gap-4">
      <p className="text-pretty text-ink-1">
        {
          "This is a different Plex account. SlipStream removes your Plex servers and stops tracking Members' access to them. Members keep their access in Plex."
        }
      </p>
      {error !== undefined && <SignalLine tone="fail">{error}</SignalLine>}
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onBack}>Back</Button>
        <Button look="fail" busy={busy} onClick={() => void replace()}>
          Use this Plex account
        </Button>
      </div>
    </div>
  );
}

/**
 * Waits for the Admin to approve the sign-in in the Plex tab, and asks before
 * a sign-in as a different Plex account replaces SlipStream's.
 */
export function WaitingPart({
  pin,
  onBack,
  onSettled,
}: {
  pin: PlexPin;
  onBack: () => void;
  onSettled: (state: PinEnd) => void;
}) {
  const [other, setOther] = useState(false);
  const settled = useCallback(
    (state: Exclude<PinState, "waiting">) => {
      if (state === "other_account") {
        setOther(true);
      } else {
        onSettled(state);
      }
    },
    [onSettled],
  );
  if (other) {
    return <OtherAccountPart pin={pin} onBack={onBack} onSettled={onSettled} />;
  }
  return (
    <div className="flex flex-col gap-4">
      <p className="flex items-center gap-2 text-pretty text-ink-1">
        <Spinner />
        Waiting for you to sign in to Plex in the other tab.
      </p>
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {"Closed the tab, or it didn't open? "}
        <a
          href={pin.authUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="font-medium text-ink-2 underline hover:text-ink-1"
        >
          Open the Plex sign-in again
        </a>
      </p>
      <Suspense fallback={null}>
        <WatchPin pinId={pin.pinId} onSettled={settled} />
      </Suspense>
      <div className="flex justify-end">
        <Button onClick={onBack}>Back</Button>
      </div>
    </div>
  );
}
