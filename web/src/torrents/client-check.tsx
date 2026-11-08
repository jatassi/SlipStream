import type { ReactNode } from "react";

import { useCheckHealthCategory } from "../api/gen/query/health/health";
import { Checked, useCheckAnswered } from "../health/checked";
import { Button } from "../ui/button";

/**
 * A red card's Check now, which checks every external client through the
 * Download Clients category, then its answer. More actions follow it.
 */
export function CheckNow({ children }: { children: ReactNode }) {
  const check = useCheckHealthCategory({ mutation: useCheckAnswered() });
  return (
    <>
      {check.data !== undefined && (
        <div className="text-fail-dim">
          <Checked unreached={check.data.unreached} />
        </div>
      )}
      <div className="flex flex-wrap justify-end gap-2">
        <Button
          look="fail"
          busy={check.isPending}
          onClick={() => {
            check.mutate({ category: "download-clients" });
          }}
        >
          Check now
        </Button>
        {children}
      </div>
    </>
  );
}
