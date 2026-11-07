import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { IndexersIndexer } from "../api/gen/model";
import { updateIndexer } from "../api/gen/query/indexers/indexers";
import { adoptIndexer } from "../live/adopt";
import { Switch } from "../ui/switch";
import { toast } from "../ui/toast";

export type SwitchName = "on" | "automaticSearch" | "rss" | "allowFailover";

export const switchLabels: { readonly [S in SwitchName]: string } = {
  on: "On",
  automaticSearch: "Automatic search",
  rss: "RSS",
  allowFailover: "Allow failover",
};

/** One of an Indexer's switches, saved through PATCH as it flips. */
export function IndexerSwitch({ indexer, name }: { indexer: IndexersIndexer; name: SwitchName }) {
  const queryClient = useQueryClient();
  const [pending, setPending] = useState<boolean>();
  const flip = async (next: boolean) => {
    setPending(next);
    try {
      adoptIndexer(queryClient, await updateIndexer(indexer.id, { [name]: next }));
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setPending(undefined);
    }
  };
  return (
    <Switch
      label={switchLabels[name]}
      checked={pending ?? indexer[name]}
      disabled={pending !== undefined}
      onCheckedChange={(next) => void flip(next)}
    />
  );
}
