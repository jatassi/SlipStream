import { useNavigate } from "@tanstack/react-router";

import type { FilesNotFoundNeed, FilesUpgradeNeed, WorksContentType } from "../../api/gen/model";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { Button } from "../../ui/button";
import { Callout } from "../../ui/callout";
import { useClock } from "../../ui/clock";
import { List, ListRow } from "../../ui/list";
import { notFoundWords } from "../../grabs/search-words";
import { useWorkSearching } from "../../grabs/searching-query";
import { type Naming, notFoundSubject, upgradeWords } from "./need-words";

export function Upgrades({
  naming,
  needs,
}: {
  naming: Naming;
  needs: readonly FilesUpgradeNeed[];
}) {
  const search = useStartSearch();
  return (
    <List label="Upgrades available">
      {needs.map((need) => (
        <ListRow
          key={`${need.itemIds.join(" ")} ${need.edition}`}
          title={upgradeWords(naming, need)}
          meta="A better release would replace the file."
          trailing={
            <span className="flex gap-2">
              <Button
                size="small"
                busy={search.isPending}
                onClick={() => {
                  search.start(naming.work.id, itemsSearched(naming), {
                    levelIds: [],
                    itemIds: need.itemIds,
                  });
                }}
              >
                Search
              </Button>
              <ManualSearchButton
                workId={naming.work.id}
                scope={{ itemIds: need.itemIds }}
                size="small"
              />
            </span>
          }
        />
      ))}
    </List>
  );
}

const itemsSearched = (naming: Naming) =>
  naming.type.items === "one" ? { name: naming.work.label } : { item: naming.type.item };

/** Missing Items whose group's last search found nothing, with when SlipStream searches them next, as the hero reads that group. */
export function NotFound({
  naming,
  needs,
}: {
  naming: Naming;
  needs: readonly FilesNotFoundNeed[];
}) {
  const search = useStartSearch();
  const clock = useClock();
  const { groups } = useWorkSearching(naming.work.id);
  return (
    <List label="Not found">
      {needs.map((need) => {
        const group = groups.find(({ key }) => key === need.groupKey);
        return (
          <ListRow
            key={`${need.itemIds.join(" ")} ${need.edition}`}
            title={notFoundSubject(naming, need)}
            meta={group === undefined ? undefined : notFoundWords(group, clock)}
            trailing={
              <span className="flex gap-2">
                <Button
                  size="small"
                  busy={search.isPending}
                  onClick={() => {
                    search.start(naming.work.id, itemsSearched(naming), {
                      levelIds: [],
                      itemIds: need.itemIds,
                    });
                  }}
                >
                  Search
                </Button>
                <ManualSearchButton
                  workId={naming.work.id}
                  scope={{ itemIds: need.itemIds }}
                  size="small"
                />
              </span>
            }
          />
        );
      })}
    </List>
  );
}

export function NoRoot({ type }: { type: WorksContentType }) {
  const navigate = useNavigate();
  return (
    <Callout
      tone="warn"
      chip="No root"
      subject={`This ${type.work.one.toLowerCase()} has no library root`}
      action={{
        label: "Open Library roots",
        onClick: () =>
          void navigate({
            to: "/settings/$section",
            params: { section: "library-roots" },
          }),
      }}
    >
      Its downloads wait until it has one. Add a library root, or choose one in Edit.
    </Callout>
  );
}
