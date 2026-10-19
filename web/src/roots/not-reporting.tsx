import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { HealthNotReporting, SettingAmount } from "../api/gen/model";
import { deleteHealthVolume } from "../api/gen/query/health/health";
import { dropVolumeLevels } from "../live/adopt";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { spaceWords } from "../ui/setting-space";
import { toast } from "../ui/toast";

const levelWords = (amount: SettingAmount | null): string =>
  amount === null ? "the global level" : spaceWords(amount);

function NotReportingRow({ volume }: { volume: HealthNotReporting }) {
  const queryClient = useQueryClient();
  const where = `${volume.mount} on ${volume.nodeName}`;
  const remove = async () => {
    try {
      await deleteHealthVolume({ nodeId: volume.nodeId, mount: volume.mount });
      dropVolumeLevels(queryClient, volume);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    }
  };
  return (
    <li className="flex flex-wrap items-center justify-between gap-3 px-4 py-3.5 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <span className="flex min-w-0 flex-col gap-0.5">
        <span>
          <Data className="break-all">{volume.mount}</Data> on {volume.nodeName}
        </span>
        <span className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          Warning below {levelWords(volume.warnBelow)}, error below {levelWords(volume.errorBelow)}
        </span>
      </span>
      <Button aria-label={`Remove the levels for ${where}`} onClick={() => void remove()}>
        Remove
      </Button>
    </li>
  );
}

/** Volumes with levels of their own that no Node reports now, kept until the Admin removes them. */
export function NotReporting({ volumes }: { volumes: readonly HealthNotReporting[] }) {
  if (volumes.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-2">
      <h3 className="font-semibold text-ink-2">Not reporting</h3>
      <p className="text-ink-3">
        These volumes have levels of their own, but no Node reports them now. Remove the ones you no
        longer need.
      </p>
      <ul
        aria-label="Volumes not reporting"
        className="rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {volumes.map((volume) => (
          <NotReportingRow key={`${volume.nodeId} ${volume.mount}`} volume={volume} />
        ))}
      </ul>
    </div>
  );
}
