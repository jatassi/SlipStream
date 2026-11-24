import type { DownloadclientsClientCard as Card, RootsRoot } from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Checkbox } from "../ui/checkbox";
import { ClientPicker, type Side } from "./client-move-dialog";
import { RootPicker } from "./destination-picker";
import { type ClientPair, type Draining, type Picks, type RootPair, clientKey } from "./drain-plan";

function RootPickers({
  pairs,
  picks,
  onPicks,
}: {
  pairs: readonly RootPair[];
  picks: Picks;
  onPicks: (picks: Picks) => void;
}) {
  const types = useListContentTypesSuspense().data.contentTypes;
  const nounOf = (root: RootsRoot) =>
    types.find(({ id }) => id === root.contentType)?.work.many.toLowerCase() ?? "Works";
  return pairs.map(({ root, options, to }) =>
    to === undefined ? (
      <p key={root.id} className="text-ink-2">
        {`No root on another Node takes ${nounOf(root)}, so ${root.path} stays.`}
      </p>
    ) : (
      <RootPicker
        key={root.id}
        label={`Move ${root.path} to`}
        roots={options}
        value={to}
        onValueChange={(rootId) => {
          onPicks({ ...picks, roots: { ...picks.roots, [root.id]: rootId } });
        }}
      />
    ),
  );
}

function ClientPickers({
  pairs,
  others,
  picks,
  onPicks,
}: {
  pairs: readonly ClientPair[];
  others: readonly Card[];
  picks: Picks;
  onPicks: (picks: Picks) => void;
}) {
  return pairs.map(({ side, name, to }) =>
    to === undefined ? null : (
      <fieldset key={clientKey(side)} className="flex flex-col gap-1">
        <legend className="mb-1 text-ink-1">{`Move ${name}'s torrents to`}</legend>
        <ClientPicker
          cards={others}
          value={to}
          onValueChange={(picked: Side) => {
            onPicks({ ...picks, clients: { ...picks.clients, [clientKey(side)]: picked } });
          }}
        />
      </fieldset>
    ),
  );
}

/** Where each of the drained Node's roots and clients goes, and whether its emptied roots go. */
export function DrainPickers({
  draining,
  roots,
  clients,
  others,
  picks,
  onPicks,
}: {
  draining: Draining;
  roots: readonly RootPair[];
  clients: readonly ClientPair[];
  others: readonly Card[];
  picks: Picks;
  onPicks: (picks: Picks) => void;
}) {
  return (
    <>
      <RootPickers pairs={roots} picks={picks} onPicks={onPicks} />
      <ClientPickers pairs={clients} others={others} picks={picks} onPicks={onPicks} />
      {draining.storage && (
        <Checkbox
          label="Delete the roots it empties"
          checked={picks.deleteEmptied}
          onCheckedChange={(deleteEmptied) => {
            onPicks({ ...picks, deleteEmptied });
          }}
        />
      )}
    </>
  );
}
