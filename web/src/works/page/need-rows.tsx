import { useNavigate } from "@tanstack/react-router";

import type { FilesUpgradeNeed } from "../../api/gen/model";
import { ManualSearchButton, useStartSearch } from "../../grabs/search";
import { Button } from "../../ui/button";
import { List, ListRow } from "../../ui/list";
import { WarningLine } from "../../ui/warning-line";
import { type Naming, upgradeWords } from "./need-words";

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
                  search.start(naming.work.id, {
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

export function NoRoot() {
  const navigate = useNavigate();
  return (
    <WarningLine
      signal="warn"
      chip="No root"
      subject="No library root for this Work"
      detail="Its downloads wait until it has one. Add a library root, or choose one in Edit."
      action={{
        label: "Open Library roots",
        onClick: () =>
          void navigate({
            to: "/settings/$section",
            params: { section: "library-roots" },
          }),
      }}
    />
  );
}
