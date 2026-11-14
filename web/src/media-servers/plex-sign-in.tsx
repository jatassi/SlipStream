import { Suspense, useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  MediaserversPlexPin as PlexPin,
  MediaserversPlexPinStateState as PinState,
} from "../api/gen/model";
import { Button } from "../ui/button";
import { Spinner } from "../ui/spinner";
import { SignalLine, WarnLine } from "./lines";
import { startPlexTab, useWatchPin } from "./use-plex-pin";

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
      {lapse !== undefined && <WarnLine text={lapseWords[lapse]} />}
      <p className="text-pretty text-ink-1">
        Plex opens in a new tab, where you sign in to your Plex account. Your password stays with
        Plex, and SlipStream lists your Plex servers here once you approve the sign-in.
      </p>
      {error !== undefined && <SignalLine level="fail" text={error} />}
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onBack}>Back</Button>
        <Button look="primary" busy={busy} onClick={() => void start()}>
          Sign in with Plex
        </Button>
      </div>
    </div>
  );
}

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

/** Waits for the Admin to approve the sign-in in the Plex tab. */
export function WaitingPart({
  pin,
  onBack,
  onSettled,
}: {
  pin: PlexPin;
  onBack: () => void;
  onSettled: (state: Exclude<PinState, "waiting">) => void;
}) {
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
        <WatchPin pinId={pin.pinId} onSettled={onSettled} />
      </Suspense>
      <div className="flex justify-end">
        <Button onClick={onBack}>Back</Button>
      </div>
    </div>
  );
}
