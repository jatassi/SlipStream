import type { DownloadclientsActed, DownloadclientsHitAndRun } from "../api/gen/model";
import { bytes } from "../format";
import { useNow } from "../tasks/use-now";
import { Data } from "../ui/data";
import { type Subject, outcomeWords } from "./actions";
import { owedWords } from "./seeding";

export type Parting = "remove" | "mark-failed";

export const list = new Intl.ListFormat("en", { type: "conjunction" });

const cancelsLead: { readonly [P in Parting]: (count: number) => string } = {
  remove: (count) =>
    count === 1
      ? "It cancels this Grab, still downloading, and its Item goes back to wanted:"
      : `It cancels ${count} Grabs still downloading, and their Items go back to wanted:`,
  "mark-failed": (count) =>
    count === 1
      ? "This Grab fails, and its Item goes back to wanted:"
      : `These ${count} Grabs fail, and their Items go back to wanted:`,
};

const parted: { readonly [P in Parting]: (who: string) => string } = {
  remove: (who) => `Removing ${who}`,
  "mark-failed": (who) => `Marking ${who} failed`,
};

const owes = (hit: DownloadclientsHitAndRun) =>
  hit.owes === undefined ? undefined : owedWords(hit.owes, hit.clears);

function HitsAndRuns({
  parting,
  hits,
  subjects,
}: {
  parting: Parting;
  hits: readonly DownloadclientsHitAndRun[];
  subjects: readonly Subject[];
}) {
  const names = new Map(subjects.map(({ id, name }) => [id, name]));
  const name = (id: string) => (subjects.length === 1 ? undefined : names.get(id));
  const groups = new Map<string | undefined, DownloadclientsHitAndRun[]>();
  for (const hit of hits) {
    groups.set(hit.tracker?.name, [...(groups.get(hit.tracker?.name) ?? []), hit]);
  }
  return [...groups].map(([tracker, owing]) => {
    const [only, ...rest] = owing;
    if (only !== undefined && rest.length === 0) {
      const owed = owes(only);
      return (
        <p key={only.torrentId} className="text-warn-ink">
          {`${parted[parting](name(only.torrentId) ?? "it")} now counts as a hit and run on ${tracker ?? "its Tracker"}.`}
          {owed !== undefined && ` It still owes ${owed}.`}
        </p>
      );
    }
    return (
      <div key={tracker ?? ""}>
        <p className="text-warn-ink">
          {`${parted[parting]("these")} now counts as a hit and run on ${tracker === undefined ? "each one's Tracker" : `${tracker} for each`}:`}
        </p>
        <ul className="mt-2 flex flex-col gap-1">
          {owing.map((hit) => (
            <li key={hit.torrentId} className="flex min-w-0 flex-col">
              <span className="text-ink-1">{name(hit.torrentId) ?? "A torrent"}</span>
              {owes(hit) !== undefined && (
                <span className="text-meta text-ink-3">{`Owes ${owes(hit)}`}</span>
              )}
            </li>
          ))}
        </ul>
      </div>
    );
  });
}

export const freesWords = (frees: number | null | undefined) =>
  frees === null || frees === undefined ? "" : ` (frees ${bytes(frees)})`;

export function Preview({
  parting,
  acted,
  subjects,
  work,
}: {
  parting: Parting;
  acted: DownloadclientsActed;
  subjects: readonly Subject[];
  work: (id: string) => string | undefined;
}) {
  const preview = acted.preview;
  const refused = outcomeWords(acted.results, subjects, {
    action: parting,
    work,
    clock: { now: useNow(60_000), zone: undefined },
  }).refused;
  const stops = (preview?.stopSeeding ?? []).map((id) => work(id) ?? "A deleted Work");
  return (
    <>
      {refused.map((sentence) => (
        <p key={sentence} className="text-warn-ink">
          {sentence}
        </p>
      ))}
      {preview !== undefined && preview.cancels.length > 0 && (
        <div>
          <p>{cancelsLead[parting](preview.cancels.length)}</p>
          <ul className="mt-2 flex flex-col gap-1">
            {preview.cancels.map((cancel) => (
              <li key={cancel.grabId} className="flex min-w-0 flex-col">
                <span className="text-ink-1">{work(cancel.workId) ?? "A deleted Work"}</span>
                <Data className="truncate text-meta text-ink-3">{cancel.release}</Data>
              </li>
            ))}
          </ul>
        </div>
      )}
      {preview !== undefined && (
        <HitsAndRuns parting={parting} hits={preview.hitAndRun} subjects={subjects} />
      )}
      {stops.length > 0 && (
        <p>
          {list.format(stops)} {stops.length === 1 ? "stops" : "stop"} seeding.
        </p>
      )}
    </>
  );
}

export function SpaceNote({ acted }: { acted: DownloadclientsActed | undefined }) {
  const preview = acted?.preview;
  if (preview === undefined) {
    return null;
  }
  if (preview.frees === null) {
    return (
      <p className="text-meta text-ink-3">
        {"The space this frees is unknown because a Node that holds it can't be reached."}
      </p>
    );
  }
  return preview.linked ? (
    <p className="text-meta text-ink-3">
      {
        "Your library shares some of these files, so the copy frees only what the library doesn't use."
      }
    </p>
  ) : null;
}
