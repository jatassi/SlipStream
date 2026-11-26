import { useSuspenseQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "@tanstack/react-router";

import { getListMyMediaServerAccessSuspenseQueryOptions } from "../../api/gen/query/mediaservers/mediaservers";
import { reconfirmWithPasskey, usePasskeySupportSoon } from "../../session/passkey";
import { ReconfirmDialog } from "../../shell/reconfirm-dialog";
import { Button } from "../../ui/button";
import type { WelcomeStepProps } from "../../welcome/steps";
import { useMemberPlexPin } from "./use-member-plex-pin";
import { WatchBlock } from "./watch-block";

declare module "@tanstack/react-router" {
  interface HistoryState {
    /** Set by portal Home's Watch card, whose Member goes back there. */
    readonly fromHome?: true;
  }
}

function Footer({ pending, next }: { pending: boolean; next: () => void }) {
  const fromHome = useLocation({ select: (location) => location.state.fromHome === true });
  const navigate = useNavigate();
  if (fromHome) {
    return (
      <Button
        onClick={() => {
          void navigate({ to: "/portal" });
        }}
      >
        Back to Home
      </Button>
    );
  }
  return pending ? (
    <Button look="ghost" onClick={next}>
      Skip for now
    </Button>
  ) : (
    <Button look="primary" onClick={next}>
      Continue
    </Button>
  );
}

export function WatchStep({ next }: WelcomeStepProps) {
  const mine = useSuspenseQuery(getListMyMediaServerAccessSuspenseQueryOptions()).data;
  const pin = useMemberPlexPin();
  const passkeys = usePasskeySupportSoon();
  return (
    <>
      {mine.rows.map((row) => (
        <WatchBlock key={row.id} row={row} mine={mine} pin={pin} />
      ))}
      <Footer pending={mine.rows.some((row) => row.pending)} next={next} />
      <ReconfirmDialog passkey={passkeys?.available === true ? reconfirmWithPasskey : undefined} />
    </>
  );
}
