import { usePrefetchQuery, useSuspenseQuery } from "@tanstack/react-query";
import { Suspense } from "react";

import type { GrabsListedRelease, GrabsListedReleaseOutcome, GrabsSearch } from "../api/gen/model";
import { getListCustomFormatsSuspenseQueryOptions } from "../api/gen/query/formats/formats";
import {
  getListGrabsSuspenseQueryOptions,
  getListSearchReleasesSuspenseQueryOptions,
} from "../api/gen/query/grabs/grabs";
import {
  getListQualitiesSuspenseQueryOptions,
  getListTargetsSuspenseQueryOptions,
} from "../api/gen/query/targets/targets";
import { Details } from "../ui/details";
import { Skeleton } from "../ui/skeleton";
import { ReleaseLine, type RunNames } from "./release-line";

const outcomeOrder: Record<GrabsListedReleaseOutcome, number> = {
  grabbed: 0,
  not_taken: 1,
  outranked: 2,
  kept: 3,
  refused: 4,
};

/** The grabbed Release first, then the ones not taken, outranked and kept, then the refused ones. */
const byOutcome = (releases: readonly GrabsListedRelease[]): GrabsListedRelease[] =>
  releases.toSorted((a, b) => outcomeOrder[a.outcome] - outcomeOrder[b.outcome]);

function useRunNames(search: GrabsSearch) {
  const queries = {
    listed: getListSearchReleasesSuspenseQueryOptions(search.id),
    formats: getListCustomFormatsSuspenseQueryOptions(),
    targets: getListTargetsSuspenseQueryOptions(),
    qualities: getListQualitiesSuspenseQueryOptions(),
    grabs: getListGrabsSuspenseQueryOptions(),
  };
  usePrefetchQuery(queries.formats);
  usePrefetchQuery(queries.targets);
  usePrefetchQuery(queries.qualities);
  usePrefetchQuery(queries.grabs);
  const listed = useSuspenseQuery(queries.listed).data;
  const formats = useSuspenseQuery(queries.formats).data;
  const targets = useSuspenseQuery(queries.targets).data;
  const qualities = useSuspenseQuery(queries.qualities).data;
  const grabs = useSuspenseQuery(queries.grabs).data;
  const targetsById = new Map(targets.targets.map((target) => [target.id, target]));
  const grab = grabs.grabs.find(({ id }) => id === search.grabId);
  const names: RunNames = {
    targets: targetsById,
    qualities: new Map(qualities.qualities.map(({ key, name }) => [key, name])),
    formats: new Map(formats.formats.map(({ id, name }) => [id, name])),
    grabbedFor: (grab?.targetIds ?? []).flatMap((id) => targetsById.get(id)?.name ?? []),
    indexerAt: new Map(listed.releases.map((release) => [release.position, release.indexerName])),
  };
  return { releases: byOutcome(listed.releases), names };
}

function SearchReleases({ search }: { search: GrabsSearch }) {
  const { releases, names } = useRunNames(search);
  if (releases.length === 0) {
    return <p className="text-ink-3">A later search of this Item replaced this list.</p>;
  }
  return (
    <ul aria-label="Releases" className="flex flex-col divide-y divide-hairline">
      {releases.map((release) => (
        <ReleaseLine key={release.position} release={release} names={names} />
      ))}
    </ul>
  );
}

/** Every Release a run saw under Details, each with its outcome in words, loaded on open. */
export function RunReleases({ search }: { search: GrabsSearch }) {
  return (
    <Details>
      <Suspense fallback={<Skeleton label="Loading the Releases" />}>
        <SearchReleases search={search} />
      </Suspense>
    </Details>
  );
}
