import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import { sentenceOf } from "../api/errors";
import type { IndexersIndexer } from "../api/gen/model";
import { deleteIndexer } from "../api/gen/query/indexers/indexers";
import { dropIndexer } from "../live/adopt";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";

/** Delete an Indexer. Its Trackers and Grab history stay. */
export function DeleteIndexerDialog({
  indexer,
  open,
  onOpenChange,
}: {
  indexer: IndexersIndexer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  return (
    <ConfirmDialog
      title={`Delete ${indexer.name}?`}
      confirmLabel="Delete"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        try {
          await deleteIndexer(indexer.id);
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        dropIndexer(queryClient, indexer.id);
        toast(`Deleted ${indexer.name}`);
        await navigate({ to: "/settings/$section", params: { section: "indexers" } });
      }}
    >
      <p>
        Searches stop asking it. Its Trackers stay, and so does the history of the Grabs it made.
      </p>
    </ConfirmDialog>
  );
}
