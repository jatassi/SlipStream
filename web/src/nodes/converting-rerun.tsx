import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { NodesNode } from "../api/gen/model";
import {
  getGetNodeConvertingQueryKey,
  rerunNodeTests,
} from "../api/gen/query/conversions/conversions";
import { refresh } from "../live/refresh";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

/** Reruns the Node's test runs, then reads its Converting section again. */
export function RerunButton({ node }: { node: NodesNode }) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  return (
    <Button
      disabled={busy}
      onClick={() => {
        setBusy(true);
        void rerunNodeTests(node.id)
          .then(() => {
            toast(`Rerunning the test runs on ${node.name}`);
            return refresh(queryClient, { queryKey: getGetNodeConvertingQueryKey(node.id) });
          })
          .catch((error: unknown) => {
            toast(`The test runs didn't start. ${sentenceOf(error)}`, { kind: "fail" });
          })
          .finally(() => {
            setBusy(false);
          });
      }}
    >
      Rerun test runs
    </Button>
  );
}
