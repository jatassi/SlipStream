import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { HealthChecked, HealthUnreached } from "../api/gen/model";
import { adoptHealthReport } from "../live/adopt";
import { SignalLine } from "../ui/signal-line";
import { toast } from "../ui/toast";

export const failed = (error: unknown) => {
  toast(sentenceOf(error), { kind: "fail" });
};

/** A check's answer becomes the page at once, ahead of health.changed. */
export function useCheckAnswered() {
  const queryClient = useQueryClient();
  return {
    onSuccess: (checked: HealthChecked) => {
      adoptHealthReport(queryClient, checked.report);
    },
    onError: failed,
  };
}

export function Checked({ unreached }: { unreached: readonly HealthUnreached[] }) {
  if (unreached.length === 0) {
    return <output className="block">Checked.</output>;
  }
  return (
    <output className="block">
      <SignalLine tone="warn">Checked. {unreached.map(({ words }) => words).join(" ")}</SignalLine>
    </output>
  );
}
