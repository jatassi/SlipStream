import type { IndexersIndexer } from "../api/gen/model";
import { SignalLine } from "../ui/signal-line";
import { toneOf } from "./words";

/** An Indexer's status sentence as Go writes it, led by its signal when it carries one. */
export function StatusLine({ indexer }: { indexer: IndexersIndexer }) {
  const tone = toneOf(indexer);
  if (tone === undefined) {
    return <p className={indexer.on ? "text-ink-2" : "text-ink-3"}>{indexer.sentence}</p>;
  }
  return <SignalLine tone={tone}>{indexer.sentence}</SignalLine>;
}
