import { Suspense, useCallback, useState } from "react";

import type {
  MediaserversAccountView as Account,
  MediaserversKindView as Kind,
  MediaserversPlexPin as PlexPin,
  MediaserversPlexPinStateState as PinState,
  MediaserversServerView as Server,
  NodesNode,
} from "../api/gen/model";
import { readPref, writePref } from "../shell/prefs";
import { Skeleton } from "../ui/skeleton";
import { ServersPart } from "./plex-servers";
import { type Lapse, SignInPart, WaitingPart } from "./plex-sign-in";
import { usePlexAccount } from "./use-plex-pin";

/** Where the Plex step stands: signing in, waiting on the Plex tab, then picking one of the account's servers. */
type PlexAt =
  | { readonly at: "start"; readonly lapse?: Lapse }
  | { readonly at: "waiting"; readonly pin: PlexPin }
  | { readonly at: "servers" };

function firstAt(account: Account | null, pending: PlexPin | null): PlexAt {
  if (pending !== null) {
    return { at: "waiting", pin: pending };
  }
  if (account === null) {
    return { at: "start" };
  }
  return account.refused ? { at: "start", lapse: "refused" } : { at: "servers" };
}

/** The Plex kind when this tab still waits on a Plex sign-in, as after a reload. */
export const pendingPlexKind = (kinds: readonly Kind[]): Kind | undefined =>
  readPref("plexPin") === null ? undefined : kinds.find((kind) => kind.signIn === "plex_pin");

export function forgetPlexPin() {
  writePref("plexPin", null);
}

type Props = {
  kind: Kind;
  nodes: readonly NodesNode[];
  onBack: () => void;
  onSaved: (server: Server) => void;
};

function SignedIn(props: Props) {
  const account = usePlexAccount();
  if (account === null) {
    throw new Error("the Plex sign-in settled with no Plex account");
  }
  return <ServersPart {...props} account={account} />;
}

/**
 * The Add sheet's Plex step. The Admin signs in to Plex in a new tab, and
 * the step then lists the account's servers. A reload while the tab is open
 * picks the waiting sign-in up again.
 */
export function PlexStep(props: Props) {
  const account = usePlexAccount();
  const [at, setAt] = useState(() => firstAt(account, readPref("plexPin")));
  const settled = useCallback((state: Exclude<PinState, "waiting">) => {
    setAt(state === "signed_in" ? { at: "servers" } : { at: "start", lapse: "expired" });
  }, []);
  if (at.at === "start") {
    return (
      <SignInPart
        lapse={at.lapse}
        onBack={props.onBack}
        onStarted={(pin) => {
          setAt({ at: "waiting", pin });
        }}
      />
    );
  }
  if (at.at === "waiting") {
    return (
      <WaitingPart
        pin={at.pin}
        onBack={() => {
          forgetPlexPin();
          props.onBack();
        }}
        onSettled={settled}
      />
    );
  }
  return (
    <Suspense fallback={<Skeleton label="Loading your Plex servers" />}>
      <SignedIn {...props} />
    </Suspense>
  );
}
