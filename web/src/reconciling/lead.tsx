import type { ReconcilingPage, ReconcilingRestore } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { RelativeTime } from "../ui/relative-time";
import { Term } from "../ui/term";
import { whenWords } from "../ui/time-words";
import { nextOf } from "./words";

function RestoreDetails({ restore }: { restore: ReconcilingRestore }) {
  return (
    <Details>
      <Facts look="well">
        <Fact label="Backup">
          <Data>{restore.backup}</Data>
        </Fact>
        <Fact label="Reconciling since">
          <RelativeTime at={restore.startedAt} form="clock" />
        </Fact>
      </Facts>
    </Details>
  );
}

/** The page's plain-language lead: what SlipStream is doing and the next step, then the machinery under Details. */
export function Lead({ page, clock }: { page: ReconcilingPage; clock: Clock }) {
  const { restore } = page;
  if (restore === null) {
    return (
      <p className="max-w-[64ch] text-ink-1">
        {page.nodes.length === 0
          ? "Every Node's torrents are matched with the library."
          : "A newly paired Node keeps every torrent until SlipStream has matched its list of torrents with the library."}
      </p>
    );
  }
  return (
    <div className="flex max-w-[64ch] flex-col gap-2">
      <p className="text-ink-1">
        SlipStream is <Term id="reconciling" /> after restoring the Backup taken{" "}
        {whenWords(restore.takenAt, "at", clock)}. Searches and removals wait until it finishes.
      </p>
      <p className="text-ink-1">{nextOf(page, clock).sentence}</p>
      <RestoreDetails restore={restore} />
    </div>
  );
}
