import { useNavigate } from "@tanstack/react-router";

import type { FilesUpgradeNeed, GrabsGrab } from "../../api/gen/model";
import { useListNodesSuspense } from "../../api/gen/query/nodes/nodes";
import { useStartSearch } from "../../grabs/search";
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
            <Button
              size="small"
              busy={search.isPending}
              onClick={() => {
                search.mutate({ workId: naming.work.id });
              }}
            >
              Search
            </Button>
          }
        />
      ))}
    </List>
  );
}

/** A Grab whose Node is away, the one reason a Grab waits today. */
export function Waiting({ grab }: { grab: GrabsGrab | undefined }) {
  const navigate = useNavigate();
  const { nodes } = useListNodesSuspense().data;
  const node = nodes.find(({ id }) => id === grab?.nodeId);
  const release = grab?.releaseTitle ?? "This Work's download";
  if (node === undefined) {
    return (
      <WarningLine
        signal="warn"
        chip="Waiting"
        subject={release}
        detail="This download has stopped until an Admin acts. Activity shows its stage."
        action={{
          label: "Open Activity",
          onClick: () => void navigate({ to: "/activity" }),
        }}
      />
    );
  }
  return (
    <WarningLine
      signal="warn"
      chip="Waiting"
      subject={`Waiting for ${node.name}`}
      detail={`Its download carries on once ${node.name} is back online.`}
      action={{
        label: `Open ${node.name}`,
        onClick: () => void navigate({ to: "/nodes/$nodeId", params: { nodeId: node.id } }),
      }}
    />
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
