import { useState } from "react";

import type { WorksContentType } from "../api/gen/model";
import { sentenceOf } from "../api/errors";
import { cancelQueuedSearches } from "../api/gen/query/grabs/grabs";
import { countWords } from "../format";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { useSearchSeveral } from "./several";
import type { WantedFilter } from "./wanted-query";
import { worksNounOf, worksWords } from "./wanted-words";

const queuedSearches = { one: "queued search", many: "queued searches" } as const;

/** Empties the bulk lane. A search already running finishes. */
export function CancelQueued() {
  const [busy, setBusy] = useState(false);
  const cancel = async () => {
    setBusy(true);
    try {
      const { canceled } = await cancelQueuedSearches({ lane: "bulk" });
      toast(`Canceled ${countWords(canceled, queuedSearches)}.`);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  return (
    <Button busy={busy} onClick={() => void cancel()}>
      Cancel queued searches
    </Button>
  );
}

/** Searches every Work the filter lists, labelled with their count in `type`'s noun. */
export function SearchAll({
  filter,
  type,
  total,
}: {
  filter: WantedFilter;
  type: WorksContentType | undefined;
  total: number;
}) {
  const several = useSearchSeveral(type);
  return (
    <>
      <Button
        look="primary"
        busy={several.busy}
        disabled={total === 0}
        onClick={() => {
          several.ask(
            {
              wanted: {
                ...(filter.contentType !== undefined && { contentType: filter.contentType }),
                reasons: [...filter.reasons],
              },
            },
            total,
          );
        }}
      >
        {`Search ${worksWords(total, worksNounOf(type))}`}
      </Button>
      {several.dialog}
    </>
  );
}
