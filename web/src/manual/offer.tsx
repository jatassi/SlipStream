import { use } from "react";

import type { GrabsOffer, GrabsWordedReason } from "../api/gen/model";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import type { Grabbing } from "./grabbing";
import { Overrides, ReleaseName, Result } from "./parts";
import { pickOf } from "./picks";
import { ReadingContext } from "./reading";
import { ReleaseDetails } from "./release-details";

/** "Grab for 1080p" when the Grab replaces a file it isn't better than, else "Grab". */
function grabWords(
  lifts: readonly GrabsWordedReason[],
  targets: ReadonlyMap<string, string>,
): string {
  const forced = lifts.find(({ code }) => code === "not_an_upgrade");
  const name = forced?.targetId === undefined ? undefined : targets.get(forced.targetId);
  return name === undefined ? "Grab" : `Grab for ${name}`;
}

/**
 * One Release of All Releases: what a Grab of it alone gets, or why it
 * can't be grabbed now, and a Grab of its own.
 */
export function Offer({ offer, grab }: { offer: GrabsOffer; grab: Grabbing }) {
  const { release } = offer;
  const { targets } = use(ReadingContext);
  const key = `release:${release.id}`;
  const result = grab.results.get(key);
  const send = (cancelGrabId?: string) => {
    grab.post([{ key, pick: pickOf({ ...offer.grab, release }, undefined, cancelGrabId) }]);
  };
  return (
    <li className="flex flex-col items-start gap-1.5 py-3">
      <ReleaseName release={release} />
      {offer.grab.outcome !== undefined && <p>{offer.grab.outcome}</p>}
      {offer.grab.refusal !== undefined && <p>{offer.grab.refusal}</p>}
      {offer.grab.waits !== undefined && <p>{offer.grab.waits}</p>}
      {release.blocked !== null && <p>{release.blocked}</p>}
      {offer.grab.cancels.map((held) => (
        <p key={held.grabId}>{held.words}</p>
      ))}
      <Overrides lifts={offer.grab.lifts} edition={offer.grab.edition} />
      {result === undefined ? (
        offer.grab.grabbable && (
          <Button
            size="small"
            busy={grab.busy}
            onClick={() => {
              send();
            }}
          >
            {release.blocked === null
              ? grabWords(offer.grab.lifts, targets)
              : "Remove from Blocklist and grab"}
          </Button>
        )
      ) : (
        <Result result={result} busy={grab.busy} onCancelHeld={send} />
      )}
      <Details>
        <ReleaseDetails release={release} />
      </Details>
    </li>
  );
}
