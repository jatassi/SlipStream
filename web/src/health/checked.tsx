import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { HealthChecked, HealthUnreached } from "../api/gen/model";
import { adoptHealthReport } from "../live/adopt";
import { Icon } from "../ui/icon";
import { signals } from "../ui/signal-chip";
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
    <output className="flex items-start gap-1.5">
      <Icon icon={signals.warn.icon} className="mt-[0.2em] size-4 shrink-0 text-warn" />
      <span>Checked. {unreached.map(({ words }) => words).join(" ")}</span>
    </output>
  );
}
