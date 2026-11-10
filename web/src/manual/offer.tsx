import type { GrabsOffer } from "../api/gen/model";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import type { Grabbing } from "./grabbing";
import { Overrides, ReleaseName, Result } from "./parts";
import { pickOf } from "./picks";
import { ReleaseDetails } from "./release-details";

/**
 * One Release of All Releases: what a Grab of it alone gets, or why it
 * can't be grabbed now, and a Grab of its own.
 */
export function Offer({ offer, grab }: { offer: GrabsOffer; grab: Grabbing }) {
  const { release } = offer;
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
            {release.blocked === null ? "Grab" : "Remove from Blocklist and grab"}
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
