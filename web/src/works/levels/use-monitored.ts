import { useQueryClient } from "@tanstack/react-query";

import type { WorksWorkMonitored } from "../../api/gen/model";
import { useSetWorkMonitored } from "../../api/gen/query/works/works";
import { noteMonitored } from "../../live/monitored";
import { toast } from "../../ui/toast";

/** Monitors levels and Items of one Work, and keeps the cached Work in step once the server has it. */
export function useMonitorParts(workId: string) {
  const queryClient = useQueryClient();
  const { mutate } = useSetWorkMonitored({
    mutation: {
      onSuccess: (_none, { data }) => {
        noteMonitored(queryClient, workId, data);
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
  return (change: WorksWorkMonitored) => {
    mutate({ workId, data: change });
  };
}
