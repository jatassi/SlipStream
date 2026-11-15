import type { IndexersIndexerCatalog } from "../../api/gen/model";
import { useGetIndexerCatalogSuspense } from "../../api/gen/query/indexers/indexers";
import { clock } from "../../format";
import { Details } from "../../ui/details";
import { Section } from "../../ui/section";
import { RefreshNow } from "../add/not-yet";
import { Cadences } from "./cadences";
import { CatalogAddress } from "./catalog-address";
import { CustomDefinitions } from "./custom-definitions";

function refreshWords(catalog: IndexersIndexerCatalog): string {
  const { refreshedAt, nextRefreshAt, notice } = catalog;
  if (notice !== null || refreshedAt === null) {
    return notice ?? "SlipStream hasn't downloaded the Indexer catalog yet.";
  }
  const last = `SlipStream last refreshed the Indexer catalog at ${clock(refreshedAt)}.`;
  return nextRefreshAt === null ? last : `${last} It refreshes again at ${clock(nextRefreshAt)}.`;
}

/** When the catalog refreshed, Refresh now, the custom definitions, and under Details where it downloads from, how often, and how often RSS sync runs. */
export function CatalogSection() {
  const catalog = useGetIndexerCatalogSuspense().data;
  return (
    <Section title="Indexer catalog">
      <div className="flex flex-col gap-5">
        <Details summary={refreshWords(catalog)} action={<RefreshNow />}>
          <div className="mt-2 flex w-full flex-col gap-4">
            <CatalogAddress key={catalog.address} catalog={catalog} />
            <Cadences />
          </div>
        </Details>
        <CustomDefinitions catalog={catalog} />
      </div>
    </Section>
  );
}
