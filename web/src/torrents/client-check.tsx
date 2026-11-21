import type { ReactNode } from "react";

import { useCheckHealthCategory } from "../api/gen/query/health/health";
import { Checked, useCheckAnswered } from "../health/checked";
import type { CalloutAction } from "../ui/callout";

/**
 * A red card's Check now, which checks every external client through the
 * Download Clients category, and the answer it shows once checked.
 */
export function useCheckNow(): { action: CalloutAction; answer: ReactNode } {
  const check = useCheckHealthCategory({ mutation: useCheckAnswered() });
  return {
    action: {
      label: "Check now",
      busy: check.isPending,
      onClick: () => {
        check.mutate({ category: "download-clients" });
      },
    },
    answer: check.data === undefined ? null : <Checked unreached={check.data.unreached} />,
  };
}
