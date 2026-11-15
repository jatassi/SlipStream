import type { GrabsCellKey } from "../../api/gen/model";
import { useGrabWaiting } from "../../api/gen/query/grabs/grabs";
import { countWords } from "../../format";
import { Button, type ButtonSize } from "../../ui/button";
import { toast } from "../../ui/toast";

const releases = { one: "Release", many: "Releases" };

/** Ends a Work's waits now, those of `cells` or every one, and toasts how many Releases it grabs. */
export function useGrabNow(workId: string) {
  const grab = useGrabWaiting({
    mutation: {
      onSuccess: ({ grabbing }) => {
        toast(
          grabbing === 0
            ? "Nothing is waiting. The wait already ended."
            : `Grabbing ${countWords(grabbing, releases)}`,
        );
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
  return {
    busy: grab.isPending,
    grab: (cells?: readonly GrabsCellKey[]) => {
      grab.mutate({ workId, data: cells === undefined ? {} : { cells: [...cells] } });
    },
  };
}

export function GrabNowButton({
  workId,
  cells,
  size = "default",
}: {
  workId: string;
  cells?: readonly GrabsCellKey[];
  size?: ButtonSize;
}) {
  const now = useGrabNow(workId);
  return (
    <Button
      size={size}
      busy={now.busy}
      onClick={() => {
        now.grab(cells);
      }}
    >
      Grab now
    </Button>
  );
}
