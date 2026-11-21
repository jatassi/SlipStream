import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import type { Place } from "../roots/use-place";
import { Page } from "../shell/page";
import { Button } from "../ui/button";
import { List } from "../ui/list";
import { FolderRow } from "./folder-row";
import { useSave } from "./folder-save";
import { type Row, leadOf, useRows, useStandings } from "./folder-standing";
import { SignalLine } from "../ui/signal-line";

/**
 * Finding the old Controller's folders: each library root at its old path
 * with the root edit's check, a folder to give for one that isn't there,
 * Save all and Later.
 */
export function FoldersStep() {
  const rows = useRows();
  const [places, setPlaces] = useState<ReadonlyMap<string, Place>>(new Map());
  const placeOf = (row: Row): Place =>
    places.get(row.root.id) ?? { nodeId: row.root.nodeId, path: row.root.path };
  const { answers, busy, failure, save, forgetAnswer } = useSave(placeOf);
  const standings = useStandings(rows, placeOf, answers);
  const found = rows.filter((row) => standings.get(row.root.id)?.kind === "found");
  const saves = found.map((row) => ({ row }));
  return (
    <Page>
      <div className="flex max-w-[760px] flex-col gap-4">
        <p className="text-ink-1">{leadOf(rows, standings)}</p>
        <List label="Folders">
          {rows.map((row) => (
            <FolderRow
              key={row.root.id}
              row={row}
              place={placeOf(row)}
              standing={standings.get(row.root.id) ?? { kind: "checking" }}
              onPlace={(place) => {
                setPlaces((before) => new Map([...before, [row.root.id, place]]));
                forgetAnswer(row.root.id);
              }}
              onSaveAnyway={(missing) => void save([{ row, forget: missing }], rows.length === 1)}
            />
          ))}
        </List>
        {failure !== undefined && (
          <SignalLine tone="fail" announce>
            {failure}
          </SignalLine>
        )}
        <DialogActions>
          <Button disabled={busy} onClick={() => void save([], true)}>
            Later
          </Button>
          <Button
            look="primary"
            busy={busy}
            disabled={found.length === 0}
            onClick={() => void save(saves, found.length === rows.length)}
          >
            Save all
          </Button>
        </DialogActions>
      </div>
    </Page>
  );
}
