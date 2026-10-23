import { useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import type { HealthVolume, HealthVolumeInputBody } from "../api/gen/model";
import { updateHealthVolume } from "../api/gen/query/health/health";
import { adoptVolumeLevels } from "../live/adopt";
import { Button } from "../ui/button";
import { Dialog, DialogClose } from "../ui/dialog";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

type Change = Pick<HealthVolumeInputBody, "warnBelow" | "errorBelow">;

export const hasOwnLevels = (volume: HealthVolume): boolean =>
  !volume.warnBelow.isDefault || !volume.errorBelow.isDefault;

function useSetLevels(volume: HealthVolume) {
  const queryClient = useQueryClient();
  return async (change: Change) => {
    adoptVolumeLevels(
      queryClient,
      await updateHealthVolume({ nodeId: volume.nodeId, mount: volume.mount, ...change }),
    );
  };
}

/** A volume's own Warning and Error levels, each following the global one until changed. */
export function VolumeLevelsDialog({
  volume,
  open,
  onOpenChange,
}: {
  volume: HealthVolume;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const set = useSetLevels(volume);
  const own = hasOwnLevels(volume);
  const where = `${volume.name} on ${volume.nodeName}`;
  const follow = async () => {
    try {
      await set({ warnBelow: null, errorBelow: null });
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    }
  };
  return (
    <Dialog title={`Warning levels for ${volume.name}`} open={open} onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-4">
        <p className="text-ink-2">
          {own
            ? `${where} has its own levels.`
            : `${where} follows the global levels until you change one here.`}
        </p>
        <SettingGroup>
          <SettingField field={volume.warnBelow} onSave={(value) => set({ warnBelow: value })} />
          <SettingField field={volume.errorBelow} onSave={(value) => set({ errorBelow: value })} />
        </SettingGroup>
        <div className="flex flex-wrap justify-end gap-2">
          {own && <Button onClick={() => void follow()}>Follow the global levels</Button>}
          <DialogClose>
            <Button look="primary">Done</Button>
          </DialogClose>
        </div>
      </div>
    </Dialog>
  );
}
