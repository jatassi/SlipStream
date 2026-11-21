import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import type { GrabsWantedRow } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { openManualSearch } from "../manual/sheet-state";
import { List, ListRow } from "../ui/list";
import { Artwork } from "../ui/artwork";
import { Sheet, SheetCancel, SheetGroup, SheetItem } from "../ui/sheet";
import { type RowSearch, useRowSearch, workLink } from "./row-search";
import { phoneLine } from "./wanted-words";

function RowSheet({
  row,
  search,
  onClose,
}: {
  row: GrabsWantedRow | undefined;
  search: RowSearch;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  return (
    <Sheet
      label={row === undefined ? "Wanted" : `Actions for ${row.work.label}`}
      open={row !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {row !== undefined && (
        <SheetGroup>
          <SheetItem
            onClick={() => {
              search.start(row);
            }}
          >
            Search
          </SheetItem>
          <SheetItem
            onClick={() => {
              openManualSearch({ workId: row.work.id });
            }}
          >
            Manual search
          </SheetItem>
          <SheetItem
            onClick={() => {
              void navigate(workLink(row));
            }}
          >
            Open Work
          </SheetItem>
        </SheetGroup>
      )}
      <SheetCancel />
    </Sheet>
  );
}

/** Wanted's rows on a phone: two-line rows that each open an action sheet. */
export function WantedList({ rows, clock }: { rows: readonly GrabsWantedRow[]; clock: Clock }) {
  const search = useRowSearch();
  const [open, setOpen] = useState<GrabsWantedRow>();
  return (
    <>
      <List label="Wanted">
        {rows.map((row) => (
          <ListRow
            key={row.work.id}
            title={row.work.label}
            meta={phoneLine(row, clock)}
            leading={<Artwork use="poster" size="row" url={row.work.posterUrl ?? undefined} />}
            render={
              // oxlint-disable-next-line jsx-a11y/control-has-associated-label -- ListRow fills the button with the Work and its line
              <button
                type="button"
                onClick={() => {
                  setOpen(row);
                }}
              />
            }
          />
        ))}
      </List>
      <RowSheet
        row={open}
        search={search}
        onClose={() => {
          setOpen(undefined);
        }}
      />
    </>
  );
}
