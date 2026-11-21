import type { ServingWorker } from "../../api/gen/model";
import { countWords } from "../../format";
import { EntryPage } from "../../shell/entry-page";
import { useClock } from "../../ui/clock";
import { whenWords } from "../../ui/time-words";
import { PairAgain } from "./pair-again";

const torrents = { one: "torrent", many: "torrents" } as const;

function andList(words: readonly string[]): string {
  return words.length < 2
    ? words.join("")
    : `${words.slice(0, -1).join(", ")} and ${words.slice(-1).join("")}`;
}

// Anyone who reaches the Worker may see the card, so it counts and never
// names what anyone requested.
function holdingLine(holding: readonly string[], seeding: number): string {
  const seeds = countWords(seeding, torrents);
  return holding.length === 0
    ? `Seeding ${seeds}.`
    : `Roles: ${andList(holding)} · seeding ${seeds}.`;
}

/** A Serving Worker's status card while it can't reach its Controller, and the way to pair it again. */
export function UnreachableCard({ data }: { data: ServingWorker }) {
  const { name, controller, since } = data;
  const clock = useClock();
  const when = since === undefined ? "" : ` ${whenWords(since, "since", clock)}`;
  return (
    <EntryPage title={`${name} hasn't been able to reach ${controller}${when}.`}>
      <p>{`${name} keeps seeding and finishes what it started.`}</p>
      <p>{holdingLine(data.holding, data.seeding)}</p>
      <PairAgain name={name} />
    </EntryPage>
  );
}
