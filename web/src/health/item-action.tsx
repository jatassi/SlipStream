import { Link } from "@tanstack/react-router";

import type { HealthAction, HealthCategoryID, HealthItem } from "../api/gen/model";
import { useCheckHealthItem } from "../api/gen/query/health/health";
import { useRunTask } from "../api/gen/query/tasks/tasks";
import { Button } from "../ui/button";
import { Checked, failed, useCheckAnswered } from "./checked";

type Act =
  | { kind: "check" }
  | { kind: "run"; taskId: string }
  | { kind: "link"; href: string; label: string };

function actOf({ kind, taskId, href, label }: HealthAction): Act {
  if (kind === "check") {
    return { kind };
  }
  if (kind === "run" && taskId !== undefined) {
    return { kind, taskId };
  }
  if (kind === "link" && href !== undefined && label !== undefined) {
    return { kind, href, label };
  }
  throw new Error(`a ${kind} action came without what it needs`);
}

export function ItemAction({
  category,
  item,
  action,
}: {
  category: HealthCategoryID;
  item: HealthItem;
  action: HealthAction;
}) {
  const act = actOf(action);
  if (act.kind === "check") {
    return <CheckItem category={category} item={item} />;
  }
  if (act.kind === "run") {
    return <RunItem item={item} taskId={act.taskId} />;
  }
  return (
    <Button look={item.level} nativeButton={false} render={<Link to={act.href} />}>
      {act.label}
    </Button>
  );
}

function CheckItem({ category, item }: { category: HealthCategoryID; item: HealthItem }) {
  const check = useCheckHealthItem({ mutation: useCheckAnswered() });
  return (
    <>
      <Button
        look={item.level}
        busy={check.isPending}
        aria-label={`Check now: ${item.subject}`}
        onClick={() => {
          check.mutate({ category, key: item.key });
        }}
      >
        Check now
      </Button>
      {check.data !== undefined && <Checked unreached={check.data.unreached} />}
    </>
  );
}

function RunItem({ item, taskId }: { item: HealthItem; taskId: string }) {
  const run = useRunTask({ mutation: { onError: failed } });
  return (
    <>
      <Button
        look={item.level}
        busy={run.isPending}
        aria-label={`Run now: ${item.subject}`}
        onClick={() => {
          run.mutate({ taskId });
        }}
      >
        Run now
      </Button>
      {run.isSuccess && <output className="block">Started.</output>}
    </>
  );
}
