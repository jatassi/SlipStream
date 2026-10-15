import { useQuery } from "@tanstack/react-query";

import { getGetSystemQueryKey, getSystem } from "../../api/gen/query/system/system";

export function Version() {
  const system = useQuery({
    queryKey: getGetSystemQueryKey(),
    queryFn: ({ signal }) => getSystem({ signal }),
  }).data;
  return (
    <p className="py-1 text-center text-meta text-ink-3">
      {system === undefined ? "SlipStream" : `SlipStream ${system.build.version ?? "unstamped"}`}
    </p>
  );
}
