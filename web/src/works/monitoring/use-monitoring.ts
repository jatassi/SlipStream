import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { setWorksMonitoring } from "../../api/gen/query/works/works";
import { type MonitoringChoice, monitoringBody, previewOf } from "./choice";

/** The choice's preview over the Works, refetched as the choice changes; nothing is written. */
export const useMonitoringPreview = (workIds: readonly string[], choice: MonitoringChoice) =>
  useQuery({
    queryKey: ["works", "monitoring", "preview", workIds, choice],
    queryFn: async ({ signal }) =>
      previewOf(await setWorksMonitoring(monitoringBody(workIds, choice, true), { signal })),
    placeholderData: keepPreviousData,
    staleTime: 0,
    gcTime: 0,
  });
