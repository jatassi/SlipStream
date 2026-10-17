import { ArrowDown01Icon } from "@hugeicons/core-free-icons";

import { nounOf } from "../../../../format";
import { Button } from "../../../../ui/button";
import type { LogRow, LogView } from "./buffer";

const counted = new Intl.NumberFormat("en");
const lineNoun = { one: "line", many: "lines" } as const;

const linesIn = (rows: readonly LogRow[]) => rows.filter((row) => row.kind === "line").length;

function words(view: LogView): string {
  if (!view.paused) {
    const lines = linesIn(view.lines);
    return `${counted.format(lines)} ${nounOf(lines, lineNoun)}`;
  }
  if (view.dropped) {
    return `Paused, ${counted.format(view.cap)}+ new, oldest dropped`;
  }
  return `Paused, ${counted.format(linesIn(view.waiting))} new`;
}

/** How many lines the view holds, what waits while paused, and the way back to the tail. */
export function LogStatus({
  view,
  follow,
  onJump,
}: {
  view: LogView;
  follow: boolean;
  onJump: () => void;
}) {
  const away = !follow && !view.paused;
  return (
    <div className="flex min-h-7 items-center gap-2 text-meta text-ink-3">
      <output>{away ? "Not following," : words(view)}</output>
      {away && (
        <Button look="ghost" size="small" icon={ArrowDown01Icon} onClick={onJump}>
          Jump to latest
        </Button>
      )}
    </div>
  );
}
