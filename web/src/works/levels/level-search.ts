import { useMemo } from "react";

import type { FilesCell, GrabsCellKey, GrabsWorkSearching, WorksItem } from "../../api/gen/model";
import { type Clock, useNow } from "../../ui/clock";
import { behindWords, groupWords, waitingWords } from "../../grabs/search-words";
import { levelGroupKey } from "../../grabs/searching-query";

/** What a level's header shows of its search group: a state word, its sentence, and its waits. */
export type LevelSearch = {
  readonly word: string | undefined;
  /** The full sentence the state word opens; undefined when the word says it all. */
  readonly sentence: string | undefined;
  readonly searching: boolean;
  /** The level's cells waiting for a better Release, which Grab now ends. */
  readonly waits: readonly GrabsCellKey[];
};

/** Each level's search state, by level id: Searching over Queued over Waiting. */
export function levelSearches(
  items: readonly WorksItem[],
  cells: readonly FilesCell[],
  state: GrabsWorkSearching,
  clock: Clock,
): ReadonlyMap<string, LevelSearch> {
  const levelOf = new Map(items.map((item) => [item.id, item.levelId]));
  const waiting = new Map<string, FilesCell[]>();
  for (const cell of cells) {
    const level = levelOf.get(cell.itemId);
    if (cell.waitUntil !== null && typeof level === "string") {
      waiting.set(level, [...(waiting.get(level) ?? []), cell]);
    }
  }
  const levels = new Set(
    [...levelOf.values()].filter((id): id is string => typeof id === "string"),
  );
  const out = new Map<string, LevelSearch>();
  for (const level of levels) {
    const group = state.groups.find(({ key }) => key === levelGroupKey(level));
    const waits = waiting.get(level) ?? [];
    const word = group === undefined ? undefined : groupWords[group.state];
    const soonest = waits
      .map(({ waitUntil }) => waitUntil ?? "")
      .toSorted()
      .at(0);
    out.set(level, {
      word: word ?? (soonest === undefined ? undefined : "Waiting"),
      sentence:
        group?.state === "queued"
          ? behindWords(group.behind, clock)
          : word === undefined && soonest !== undefined
            ? `${waitingWords(soonest, clock)}.`
            : undefined,
      searching: group?.state === "searching",
      waits: waits.map(({ itemId, edition, targetId }) => ({ itemId, edition, targetId })),
    });
  }
  return out;
}

/** Each level's search state, its times read again on each minute. */
export function useLevelSearches(
  items: readonly WorksItem[],
  cells: readonly FilesCell[],
  state: GrabsWorkSearching,
): ReadonlyMap<string, LevelSearch> {
  const now = useNow();
  return useMemo(
    () => levelSearches(items, cells, state, { now, zone: undefined }),
    [items, cells, state, now],
  );
}
