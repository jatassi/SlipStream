import { useState } from "react";

import { PlaceActions, PlaceForm } from "../roots/place-form";
import { filesWords } from "../roots/refused";
import { type Place, usePlace } from "../roots/use-place";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Dialog } from "../ui/dialog";
import { ListRow } from "../ui/list";
import { Spinner } from "../ui/spinner";
import { type Row, type Standing, useNodeNames } from "./folder-standing";

function ChangeFolder({
  row,
  place,
  onUse,
  onClose,
}: {
  row: Row;
  place: Place;
  onUse: (place: Place) => void;
  onClose: () => void;
}) {
  const editing = usePlace(place, row.root.id);
  return (
    <PlaceForm
      editing={editing}
      onSubmit={() => {
        if (editing.usable) {
          onUse(editing.place);
        }
      }}
      lead={
        <p className="text-ink-2">
          Pick the folder that holds the {row.step.label} now, on this Controller or any paired
          Node.
        </p>
      }
    >
      <PlaceActions busy={false} failure={undefined} onCancel={onClose}>
        <Button type="submit" look="primary" disabled={!editing.usable}>
          Use this folder
        </Button>
      </PlaceActions>
    </PlaceForm>
  );
}

function StandingLine({ standing }: { standing: Standing }) {
  switch (standing.kind) {
    case "checking":
      return (
        <p className="flex items-center gap-2 text-ink-2">
          <Spinner />
          Checking the folder…
        </p>
      );
    case "found":
      return <p className="text-ink-2">The folder is there.</p>;
    case "refused":
      return (
        <p className="text-fail-ink">
          {standing.missing === null
            ? standing.sentence
            : `${filesWords(standing.missing)} ${standing.missing === 1 ? "isn't" : "aren't"} in this folder.`}
        </p>
      );
    default:
      return standing satisfies never;
  }
}

function ChangeDialog({
  row,
  place,
  onPlace,
}: {
  row: Row;
  place: Place;
  onPlace: (place: Place) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setOpen(true);
        }}
      >
        Change folder
      </Button>
      <Dialog
        title={`Find the folder of the ${row.step.label}`}
        size={680}
        open={open}
        onOpenChange={setOpen}
      >
        {open && (
          <ChangeFolder
            row={row}
            place={place}
            onUse={(next) => {
              onPlace(next);
              setOpen(false);
            }}
            onClose={() => {
              setOpen(false);
            }}
          />
        )}
      </Dialog>
    </>
  );
}

export function FolderRow({
  row,
  place,
  standing,
  onPlace,
  onSaveAnyway,
}: {
  row: Row;
  place: Place;
  standing: Standing;
  onPlace: (place: Place) => void;
  onSaveAnyway: (missing: number) => void;
}) {
  const node = useNodeNames()(place.nodeId);
  const missing = standing.kind === "refused" ? standing.missing : null;
  return (
    <ListRow
      title={row.step.label}
      wrap
      detail={
        <div className="flex flex-col gap-2 pb-1">
          <p className="text-meta text-ink-3">
            <Data>{place.path}</Data> on {node}
          </p>
          <StandingLine standing={standing} />
          <div className="flex flex-wrap gap-2 pt-1">
            <ChangeDialog row={row} place={place} onPlace={onPlace} />
            {missing !== null && (
              <Button
                size="small"
                look="warn"
                onClick={() => {
                  onSaveAnyway(missing);
                }}
              >
                {`Save anyway, forget ${filesWords(missing)}`}
              </Button>
            )}
          </div>
        </div>
      }
    />
  );
}
