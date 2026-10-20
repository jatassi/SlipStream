import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useNow } from "../tasks/use-now";
import { StatusSentence } from "../ui/status-sentence";
import { Browse, type TorrentsSearch } from "./browse";
import { NoTorrents } from "./empty";
import { useNames } from "./names";
import { Signals } from "./signals";
import { signalsOf, torrentsSentence } from "./summary";
import type { RowSeams } from "./table";

export type { TorrentsSearch } from "./browse";

const wordsTick = 60_000;

export function TorrentsView({
  search,
  onSearchChange,
  ...seams
}: RowSeams & {
  search: TorrentsSearch;
  onSearchChange: (change: Partial<TorrentsSearch>) => void;
}) {
  const list = useListTorrentsSuspense().data;
  const names = useNames();
  const clock = { now: useNow(wordsTick), zone: undefined };
  if (list.rows.length === 0) {
    return <NoTorrents />;
  }
  const signals = signalsOf(list.rows, names.node, clock);
  return (
    <>
      <div className="mb-6">
        <StatusSentence parts={torrentsSentence(list.counts, signals)} />
      </div>
      {signals.length > 0 && (
        <div className="mb-6">
          <Signals signals={signals} names={names} />
        </div>
      )}
      <Browse
        list={list}
        search={search}
        onSearchChange={onSearchChange}
        names={names}
        clock={clock}
        {...seams}
      />
    </>
  );
}
