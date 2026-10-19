import { useState } from "react";

import type { RootsMissingFiles, RootsRoot } from "../api/gen/model";
import { rehomeRoot } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { PlaceActions, PlaceForm, useSaving } from "./place-form";
import { filesWords, missingOf, placeRefusal } from "./refused";
import { MissingFiles, RehomeProgress } from "./rehome-parts";
import { type Place, samePlace, usePlace } from "./use-place";

type Found = { readonly place: Place; readonly missing: RootsMissingFiles };

function useRehome(root: RootsRoot, label: string, onSaved: (root: RootsRoot) => void) {
  const editing = usePlace({ nodeId: root.nodeId, path: root.path }, root.id);
  const { place, namer } = editing;
  const [found, setFound] = useState<Found>();
  const saving = useSaving(
    async (forget?: number) => {
      const saved = await rehomeRoot(root.id, {
        ...place,
        ...(forget !== undefined && { forget }),
      });
      toast(`The ${label} root is now ${saved.path} on ${namer.node}.`);
      onSaved(saved);
    },
    (error) => {
      const missing = missingOf(error);
      setFound(missing && { place, missing });
      return missing === undefined ? placeRefusal(error, namer) : undefined;
    },
  );
  return {
    editing,
    saving,
    missing: found !== undefined && samePlace(found.place, place) ? found.missing : undefined,
    moved: !samePlace(place, { nodeId: root.nodeId, path: root.path }),
  };
}

/** Edit a root's Node and folder: it re-homes once every recorded file is at the new place. */
export function RehomeForm({
  root,
  onSaved,
  onCancel,
}: {
  root: RootsRoot;
  onSaved: (root: RootsRoot) => void;
  onCancel: () => void;
}) {
  const type = useListContentTypesSuspense().data.contentTypes.find(
    (candidate) => candidate.id === root.contentType,
  );
  const label = type?.work.many ?? root.contentType;
  const { editing, saving, missing, moved } = useRehome(root, label, onSaved);
  return (
    <PlaceForm editing={editing} onSubmit={() => void saving.run()}>
      <p className="text-ink-3">
        Point the root at the folder that holds its files now. SlipStream checks every file it
        recorded there first, and moves nothing.
      </p>
      {saving.busy && <RehomeProgress progress={root.rehome} />}
      {!saving.busy && missing !== undefined && <MissingFiles missing={missing} label={label} />}
      <PlaceActions busy={saving.busy} failure={saving.failure} onCancel={onCancel}>
        {missing === undefined ? (
          <Button
            type="submit"
            look="primary"
            busy={saving.busy}
            disabled={!moved || !editing.usable}
          >
            Save
          </Button>
        ) : (
          <Button look="warn" busy={saving.busy} onClick={() => void saving.run(missing.count)}>
            {`Save anyway, forget ${filesWords(missing.count)}`}
          </Button>
        )}
      </PlaceActions>
    </PlaceForm>
  );
}
