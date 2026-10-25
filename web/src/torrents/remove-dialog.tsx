import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { actOnTorrents } from "../api/gen/query/downloadclients/downloadclients";
import { Checkbox } from "../ui/checkbox";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Spinner } from "../ui/spinner";
import type { Subject } from "./actions";
import { type Parting, Preview, SpaceNote, freesWords, list } from "./remove-preview";

const titles: { readonly [P in Parting]: (subjects: readonly Subject[]) => string } = {
  remove: ([only, ...rest]) =>
    only !== undefined && rest.length === 0
      ? `Remove ${only.name}?`
      : `Remove ${rest.length + 1} torrents?`,
  "mark-failed": (subjects) => `Mark ${list.format(subjects.map(({ name }) => name))} failed?`,
};

// A dry run of the same action says what it takes with it, and which torrents it refuses.
function usePreview(parting: Parting, subjects: readonly Subject[]) {
  const ids = subjects.map(({ id }) => id);
  const preview = useQuery({
    queryKey: ["torrents-preview", parting, ...ids],
    queryFn: ({ signal }) => actOnTorrents({ action: parting, ids, dryRun: true }, { signal }),
    gcTime: 0,
  });
  const refused = new Set(
    preview.data?.results.filter((result) => !result.done).map((result) => result.id) ?? [],
  );
  return { preview, going: subjects.filter(({ id }) => !refused.has(id)) };
}

type RemoveDialogProps = {
  parting: Parting;
  subjects: readonly Subject[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (subjects: readonly Subject[], deleteFiles: boolean) => Promise<void>;
  work: (id: string) => string | undefined;
};

/**
 * Asks before a remove or a mark failed, with what it takes with it: the
 * Grabs it cancels, the Works that stop seeding and the space a delete
 * frees, from a dry run of the same action.
 */
export function RemoveDialog({
  parting,
  subjects,
  open,
  onOpenChange,
  onConfirm,
  work,
}: RemoveDialogProps) {
  const { preview, going } = usePreview(parting, subjects);
  const [deleteFiles, setDeleteFiles] = useState(false);
  const frees = freesWords(preview.data?.preview?.frees);
  return (
    <ConfirmDialog
      title={titles[parting](subjects)}
      confirmLabel={parting === "remove" ? "Remove" : "Mark failed"}
      look="fail"
      confirmDisabled={going.length === 0}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => onConfirm(going, parting === "mark-failed" || deleteFiles)}
    >
      <p>
        {parting === "remove"
          ? "Files in your library stay."
          : `It is removed and its download copy deleted${frees}. Files in your library stay.`}
      </p>
      {preview.isPending && <Spinner label="Checking what this takes with it" />}
      {preview.isError && <p className="text-fail-ink">{sentenceOf(preview.error)}</p>}
      {preview.data !== undefined && (
        <Preview parting={parting} acted={preview.data} subjects={subjects} work={work} />
      )}
      {parting === "remove" && (
        <div className="flex flex-col gap-1">
          <Checkbox
            label={`Also delete the download copy${frees}`}
            checked={deleteFiles}
            onCheckedChange={setDeleteFiles}
          />
          <SpaceNote acted={preview.data} />
        </div>
      )}
    </ConfirmDialog>
  );
}
