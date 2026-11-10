import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { GrabsOffer } from "../api/gen/model";
import {
  getListManualSearchReleasesQueryKey,
  listManualSearchReleases,
} from "../api/gen/query/grabs/grabs";
import { AlertLine } from "../ui/alert-line";
import { Details } from "../ui/details";
import { Select } from "../ui/select";
import { Skeleton } from "../ui/skeleton";
import type { Grabbing } from "./grabbing";
// oxlint-disable-next-line import/max-dependencies -- All Releases reads its own list, sorts it and grabs from it
import { Offer } from "./offer";

type Key = "rank" | "score" | "quality" | "indexer" | "size" | "age" | "seeders";

const keys: readonly { value: Key; label: string }[] = [
  { value: "rank", label: "Rank" },
  { value: "score", label: "Score" },
  { value: "quality", label: "Quality" },
  { value: "indexer", label: "Indexer" },
  { value: "size", label: "Size" },
  { value: "age", label: "Age" },
  { value: "seeders", label: "Seeders" },
];

const unranked = Number.MAX_SAFE_INTEGER;
const rankOf = (offer: GrabsOffer) => offer.rank ?? unranked;
const publishedOf = (offer: GrabsOffer) =>
  offer.release.publishedAt === null ? 0 : Date.parse(offer.release.publishedAt);

/** Each column's order, best or newest first; rank breaks every tie. */
const orders: Record<Key, (a: GrabsOffer, b: GrabsOffer) => number> = {
  rank: () => 0,
  score: (a, b) => b.score - a.score,
  quality: (a, b) => b.quality - a.quality,
  indexer: (a, b) => (a.release.indexers[0] ?? "").localeCompare(b.release.indexers[0] ?? ""),
  size: (a, b) => b.release.size - a.release.size,
  age: (a, b) => publishedOf(b) - publishedOf(a),
  seeders: (a, b) => b.release.seeders - a.release.seeders,
};

const sorted = (offers: readonly GrabsOffer[], key: Key) =>
  offers.toSorted((a, b) => orders[key](a, b) || rankOf(a) - rankOf(b));

function Offers({ workId, grab }: { workId: string; grab: Grabbing }) {
  const [key, setKey] = useState<Key>("rank");
  const listed = useQuery({
    queryKey: getListManualSearchReleasesQueryKey(workId),
    queryFn: ({ signal }) => listManualSearchReleases(workId, { signal }),
    gcTime: 0,
  });
  if (listed.error !== null) {
    return <AlertLine text={sentenceOf(listed.error)} />;
  }
  if (listed.data === undefined) {
    return <Skeleton label="Loading the Releases" />;
  }
  return (
    <>
      <Select label="Sort by" options={keys} value={key} onValueChange={setKey} />
      <ul aria-label="All Releases" className="flex flex-col divide-y divide-hairline">
        {sorted(listed.data.releases, key).map((offer) => (
          <Offer key={offer.release.id} offer={offer} grab={grab} />
        ))}
      </ul>
    </>
  );
}

/** Every Release the search found, sortable, each grabbable on its own; read once opened. */
export function AllReleases({
  workId,
  count,
  grab,
}: {
  workId: string;
  count: number;
  grab: Grabbing;
}) {
  return (
    <Details label={`All Releases (${count})`}>
      <Offers workId={workId} grab={grab} />
    </Details>
  );
}
