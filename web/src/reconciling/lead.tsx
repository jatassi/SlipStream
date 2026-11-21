import type { ReconcilingPage, ReconcilingRestore } from "../api/gen/model";
import { type Clock, pastAtWords, sinceWords } from "../format";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Term } from "../ui/term";
import { nextOf } from "./words";

function RestoreDetails({ restore, since }: { restore: ReconcilingRestore; since: string }) {
  return (
    <Details>
      <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-meta">
        <dt className="text-ink-3">Backup</dt>
        <dd>
          <Data>{restore.backup}</Data>
        </dd>
        <dt className="text-ink-3">Reconciling since</dt>
        <dd>{since}</dd>
      </dl>
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
          : "A newly paired Node keeps every torrent until SlipStream matches the torrents it reports with the library."}
      </p>
    );
  }
  return (
    <div className="flex max-w-[64ch] flex-col gap-2">
      <p className="text-ink-1">
        SlipStream is <Term id="reconciling" /> after restoring the Backup taken{" "}
        {pastAtWords(restore.takenAt, clock)}. Searches and removals wait until it finishes.
      </p>
      <p className="text-ink-1">{nextOf(page, clock).sentence}</p>
      <RestoreDetails restore={restore} since={sinceWords(restore.startedAt, clock)} />
    </div>
  );
}
