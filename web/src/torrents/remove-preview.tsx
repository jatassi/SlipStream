import type { DownloadclientsActed } from "../api/gen/model";
import { bytes } from "../format";
import { useNow } from "../tasks/use-now";
import { Data } from "../ui/data";
import { type Subject, outcomeWords } from "./actions";

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
        {"The space this frees is unknown because its Node didn't respond."}
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
