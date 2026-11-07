import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { IndexersTracker } from "../api/gen/model";
import { mergeTracker } from "../api/gen/query/indexers/indexers";
import { adoptMerge } from "../live/adopt";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Select } from "../ui/select";
import { toast } from "../ui/toast";

function WhatMoves({ from, into }: { from: string; into: string }) {
  return (
    <p>
      {from}&apos;s Indexers, Grabs and torrents move to {into}, and {from} is deleted. Releases
      that reached {from} reach {into} from then on.
    </p>
  );
}

/** Merge a Tracker into another that is the same site. The other is kept. */
export function MergeTrackerDialog({
  tracker,
  others,
  onClose,
}: {
  tracker: IndexersTracker;
  others: readonly IndexersTracker[];
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const options = others.map((other) => ({ value: other.id, label: other.name }));
  const [into, setInto] = useState(options[0]?.value ?? "");
  const kept = others.find((other) => other.id === into);
  return (
    <ConfirmDialog
      title={`Merge ${tracker.name}`}
      confirmLabel="Merge"
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onConfirm={async () => {
        try {
          const saved = await mergeTracker(tracker.id, { into });
          adoptMerge(queryClient, tracker.id, saved);
          toast(`Merged ${tracker.name} into ${saved.name}`);
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
      }}
    >
      <p>
        Merge only when both Trackers are the same site, such as one site reached from the catalog
        and through a Torznab feed.
      </p>
      <Select label="Merge into" options={options} value={into} onValueChange={setInto} />
      {kept && <WhatMoves from={tracker.name} into={kept.name} />}
    </ConfirmDialog>
  );
}
