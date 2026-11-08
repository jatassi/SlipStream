import { useNow } from "../tasks/use-now";
import { StatusSentence } from "../ui/status-sentence";
import { Browse, type Seams, type TorrentsSearch } from "./browse";
import { ClientCards } from "./client-cards";
import { NoTorrents } from "./empty";
import { GoalMetPreview } from "./goal-met";
import { useNames } from "./names";
import { type Scope, useScopedTorrents } from "./scope";
import { Signals } from "./signals";
import { signalsOf, torrentsSentence } from "./summary";

export type { TorrentsSearch } from "./browse";

const wordsTick = 60_000;

export function TorrentsView({
  search,
  onSearchChange,
  scope,
  ...seams
}: Seams & {
  search: TorrentsSearch;
  onSearchChange: (change: Partial<TorrentsSearch>) => void;
  scope?: Scope | undefined;
}) {
  const list = useScopedTorrents(scope);
  const names = useNames();
  const clock = { now: useNow(wordsTick), zone: undefined };
  const cards = <ClientCards rows={list.rows} clock={clock} />;
  if (list.rows.length === 0) {
    return (
      <>
        {cards}
        {scope === undefined ? <NoTorrents /> : scope.empty}
      </>
    );
  }
  const signals = signalsOf(list.rows, names.node, clock);
  return (
    <>
      {cards}
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
        goalMet={<GoalMetPreview rows={list.rows} names={names} />}
        {...seams}
      />
    </>
  );
}
