import { useState } from "react";

import { sentenceOf } from "../api/errors";
import { cancelQueuedSearches } from "../api/gen/query/grabs/grabs";
import { countWords } from "../format";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { useSearchSeveral } from "./several";
import type { WantedFilter } from "./wanted-query";
import { worksWords } from "./wanted-words";

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

/** Searches every Work the filter lists, labelled with their count. */
export function SearchAll({ filter, total }: { filter: WantedFilter; total: number }) {
  const several = useSearchSeveral(filter.contentType);
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
        {`Search ${worksWords(total)}`}
      </Button>
      {several.dialog}
    </>
  );
}
