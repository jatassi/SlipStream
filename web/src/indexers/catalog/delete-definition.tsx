import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { IndexersCatalogEntry } from "../../api/gen/model";
import {
  deleteCustomDefinition,
  getGetIndexerCatalogQueryKey,
} from "../../api/gen/query/indexers/indexers";
import { refresh } from "../../live/refresh";
import { Button } from "../../ui/button";
import { ConfirmDialog } from "../../ui/confirm-dialog";
import { toast } from "../../ui/toast";

/** Deletes a custom definition after asking; the server refuses while an Indexer runs it. */
export function DeleteDefinition({
  entry,
  replaces,
}: {
  entry: IndexersCatalogEntry;
  replaces: boolean;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        look="ghost"
        size="small"
        aria-label={`Delete ${entry.name}`}
        onClick={() => {
          setOpen(true);
        }}
      >
        Delete
      </Button>
      <ConfirmDialog
        title={`Delete the custom definition ${entry.name}?`}
        confirmLabel="Delete"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={async () => {
          try {
            await deleteCustomDefinition(entry.id);
          } catch (error) {
            throw new Error(sentenceOf(error), { cause: error });
          }
          await refresh(queryClient, { queryKey: getGetIndexerCatalogQueryKey() });
          toast(`Deleted the custom definition ${entry.name}`);
        }}
      >
        <p>
          {replaces
            ? `Add Indexer offers the catalog's ${entry.name} again.`
            : `Add Indexer stops offering ${entry.name}.`}
        </p>
      </ConfirmDialog>
    </>
  );
}
