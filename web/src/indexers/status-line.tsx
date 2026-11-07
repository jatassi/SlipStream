import type { IndexersIndexer } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { signals } from "../ui/signal-chip";
import { toneOf } from "./words";

const inks = { fail: "text-fail-ink", warn: "text-warn" } as const;

/** An Indexer's status sentence as Go writes it, led by its signal when it carries one. */
export function StatusLine({ indexer }: { indexer: IndexersIndexer }) {
  const tone = toneOf(indexer);
  if (tone === undefined) {
    return <p className={indexer.on ? "text-ink-2" : "text-ink-3"}>{indexer.sentence}</p>;
  }
  return (
    <p className={`flex items-start gap-1.5 ${inks[tone]}`}>
      <Icon icon={signals[tone].icon} className="mt-[0.2em] size-3.5 shrink-0" />
      {indexer.sentence}
    </p>
  );
}
