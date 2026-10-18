import { Alert02Icon } from "@hugeicons/core-free-icons";

import type { TargetsQuality } from "../api/gen/model";
import { Checkbox } from "../ui/checkbox";
import { Icon } from "../ui/icon";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";

export function CinemaSources({
  draft,
  onChange,
  quality,
}: SectionProps<TargetDraft> & { quality: TargetsQuality }) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-body font-semibold text-ink-2">Cinema sources</h3>
      <Checkbox
        label={quality.name}
        checked={draft.cinema}
        onCheckedChange={(cinema) => {
          onChange({ ...draft, cinema });
        }}
      />
      <p className="flex items-center gap-1.5 text-meta text-warn-ink">
        <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
        Cinema recordings (CAM, TS, TELESYNC, HDCAM) are filmed in a theatre. Accept them only on
        purpose; they rank below every other quality.
      </p>
    </div>
  );
}
