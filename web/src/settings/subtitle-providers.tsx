import { useState } from "react";

import type { SubtitlesProvider, WorksContentType } from "../api/gen/model";
import { useListSubtitleProvidersSuspense } from "../api/gen/query/subtitles/subtitles";
import { toastRefusal, useSubtitleWrites } from "../subtitles/settings-api";
import { providersCaption } from "../subtitles/settings-words";
import { type RankedRow, RankedList } from "../ui/ranked-list";
import { Section } from "../ui/section";
import { ProviderBody } from "./subtitle-provider-row";

type Order = { readonly from: readonly SubtitlesProvider[]; readonly ids: readonly string[] };

// A move shows at once; the list the server answers with replaces it.
function useShownOrder(providers: readonly SubtitlesProvider[]) {
  const [moved, setMoved] = useState<Order>();
  const ids = moved?.from === providers ? moved.ids : providers.map((provider) => provider.id);
  return {
    shown: ids.flatMap((id) => providers.filter((provider) => provider.id === id)),
    move: (next: readonly string[]) => {
      setMoved({ from: providers, ids: next });
    },
    undo: () => {
      setMoved(undefined);
    },
  };
}

/** The Subtitle Providers in search order, each with its state, its switch and its key. */
export function SubtitleProviders({
  types,
  openKey,
}: {
  types: readonly WorksContentType[];
  openKey: (provider: SubtitlesProvider) => void;
}) {
  const { providers } = useListSubtitleProvidersSuspense().data;
  const { orderProviders } = useSubtitleWrites();
  const { shown, move, undo } = useShownOrder(providers);
  const rows: RankedRow<SubtitlesProvider>[] = shown.map((provider) => ({
    kind: "item",
    id: provider.id,
    value: provider,
    tiedAbove: false,
  }));
  return (
    <Section title="Subtitle Providers">
      <RankedList
        label="Subtitle Providers"
        caption={providersCaption}
        rows={rows}
        name={(provider) => provider.name}
        onChange={(next) => {
          const ids = next.map((row) => row.id);
          move(ids);
          void toastRefusal(
            orderProviders(ids).catch((error: unknown) => {
              undo();
              throw error;
            }),
          );
        }}
        renderRow={(provider) => (
          <ProviderBody provider={provider} types={types} openKey={openKey} />
        )}
      />
    </Section>
  );
}
