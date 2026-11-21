import type { WorksContentType } from "../../api/gen/model";
import { Callout } from "../../ui/callout";

/** Why Refresh metadata can't run on a Work its source no longer lists. */
export const goneWords = (type: WorksContentType): string =>
  `${type.source} no longer lists this ${type.work.one.toLowerCase()}. SlipStream keeps it as it is.`;

/** The warning a Work its metadata source no longer lists carries, with Delete as its one action. */
export function GoneUpstream({ type, onDelete }: { type: WorksContentType; onDelete: () => void }) {
  return (
    <Callout
      tone="warn"
      chip={`Gone from ${type.source}`}
      subject="Refresh metadata is unavailable"
      action={{ label: "Delete", onClick: onDelete }}
    >
      {goneWords(type)}
    </Callout>
  );
}
