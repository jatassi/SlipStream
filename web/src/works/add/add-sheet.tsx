import { useQuery } from "@tanstack/react-query";
import { Suspense, useEffect } from "react";

import { DialogActions } from "../../account/dialog-actions";
import { sentenceOf } from "../../api/errors";
import {
  getContentTypeSettings,
  getGetContentTypeSettingsQueryKey,
  getPreviewCatalogRecordQueryKey,
  previewCatalogRecord,
} from "../../api/gen/query/works/works";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { Dialog, DialogClose } from "../../ui/dialog";
import { Skeleton } from "../../ui/skeleton";
import { AddForm } from "./add-form";
// oxlint-disable-next-line import/max-dependencies -- the sheet reads the preview and the type's add default, and says when either fails
import { type Adding, closeAddSheet, useAddSheet } from "./sheet-state";

const loading = "Loading the preview";

export function AddSheet() {
  const state = useAddSheet();
  // A sheet left open when the session ends must not greet the next one.
  useEffect(() => closeAddSheet, []);
  if (state.shown === "none") {
    return null;
  }
  const { type, candidate } = state.adding;
  return (
    <Dialog
      title={`Add ${candidate.label}`}
      size={680}
      open={state.shown === "open"}
      onOpenChange={(open) => {
        if (!open) {
          closeAddSheet();
        }
      }}
    >
      <Suspense fallback={<Skeleton label={loading} />}>
        <Previewing key={`${type.id}/${candidate.key}`} adding={state.adding} />
      </Suspense>
    </Dialog>
  );
}

function Previewing({ adding: { type, candidate } }: { adding: Adding }) {
  const preview = useQuery({
    queryKey: getPreviewCatalogRecordQueryKey(type.id, candidate.key),
    queryFn: ({ signal }) => previewCatalogRecord(type.id, candidate.key, { signal }),
  });
  // The add default for monitoring changes with no live event, so each sheet reads it again.
  const settings = useQuery({
    queryKey: getGetContentTypeSettingsQueryKey(type.id),
    queryFn: ({ signal }) => getContentTypeSettings(type.id, { signal }),
    staleTime: 0,
    refetchOnMount: "always",
  });
  const error = preview.error ?? settings.error;
  if (error !== null) {
    return (
      <div className="flex flex-col gap-4">
        <AlertLine text={sentenceOf(error)} />
        <DialogActions>
          <DialogClose>
            <Button>Close</Button>
          </DialogClose>
        </DialogActions>
      </div>
    );
  }
  if (preview.data === undefined || settings.data === undefined || settings.isFetching) {
    return <Skeleton label={loading} />;
  }
  return <AddForm type={type} work={preview.data.work} settings={settings.data.settings} />;
}
