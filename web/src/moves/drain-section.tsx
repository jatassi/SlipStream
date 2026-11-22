import { type Ref, useImperativeHandle, useState } from "react";

import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useListRootsSuspense, useListVolumesSuspense } from "../api/gen/query/roots/roots";
import { Checkbox } from "../ui/checkbox";
import { DrainPickers } from "./drain-pickers";
import { type Draining, type Picks, askOf, pairsOf, whatWords } from "./drain-plan";
import { DrainPreview, startDrain } from "./drain-preview";
import { useMovePreview } from "./use-move-preview";

/** What the Review's Apply runs after the change applies: the Move, when the Admin asked for it. */
export type DrainMove = { readonly start: () => Promise<void> };

const unpicked: Picks = { roots: {}, clients: {}, deleteEmptied: true };

/**
 * The drain Review's offer to move what the Node keeps: one destination
 * root per root and one destination client per client, previewed in
 * place. The Review's Apply changes the Roles, then starts the Move.
 */
export function DrainSection({ draining, ref }: { draining: Draining; ref: Ref<DrainMove> }) {
  const listed = {
    roots: useListRootsSuspense().data.roots,
    volumes: useListVolumesSuspense().data.nodes,
    cards: useListDownloadClientsSuspense().data.cards,
  };
  const [on, setOn] = useState(false);
  const [picks, setPicks] = useState(unpicked);
  const { roots, clients, others } = pairsOf(draining, picks, listed);
  const ask = on ? askOf(draining, roots, clients, picks.deleteEmptied) : undefined;
  const previewed = useMovePreview(ask && { kind: "ask", ask });
  useImperativeHandle(
    ref,
    () => ({
      start: () =>
        ask === undefined ? Promise.resolve() : startDrain(previewed, draining.node.name),
    }),
    [ask, previewed, draining.node.name],
  );
  return (
    <section aria-label="Move" className="flex flex-col gap-4 border-t border-hairline pt-4">
      <Checkbox
        label={`Also move ${draining.node.name}'s ${whatWords(draining)}`}
        checked={on}
        onCheckedChange={setOn}
      />
      {on && (
        <>
          <DrainPickers
            draining={draining}
            roots={roots}
            clients={clients}
            others={others}
            picks={picks}
            onPicks={setPicks}
          />
          <DrainPreview preview={previewed.preview} />
        </>
      )}
    </section>
  );
}
