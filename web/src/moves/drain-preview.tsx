import { sentenceOf } from "../api/errors";
import type { MovesMovePreview } from "../api/gen/model";
import { countWords } from "../format";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";
import { toast } from "../ui/toast";
import { downloadingWords } from "./drain-plan";
import { told } from "./preview-dialog";
import { Plan } from "./preview-plan";
import type { useMovePreview } from "./use-move-preview";

type Previewed = ReturnType<typeof useMovePreview>;

/**
 * Starts the drain's Move once the change applied, telling what started.
 * A Move that doesn't start leaves the change applied and says why.
 */
export async function startDrain({ preview, start }: Previewed, node: string) {
  try {
    const shown = preview.data ?? (await preview.refetch({ throwOnError: true })).data;
    if (shown === undefined || shown.movable === 0) {
      return;
    }
    const started = await start(shown);
    told(
      started,
      `Started ${countWords(started.started, { one: "Move", many: "Moves" })} from ${node}.`,
    );
  } catch (error) {
    toast(`Applied the change, but the Move didn't start. ${sentenceOf(error)}`, { kind: "fail" });
  }
}

const homedWords = (plan: MovesMovePreview) =>
  plan.homed > 0
    ? `${countWords(plan.homed, { one: "Work", many: "Works" })} without files ${plan.homed === 1 ? "goes" : "go"} into the new roots too.`
    : undefined;

/** The drain's preview: its plan, and the torrents that move once they finish. */
export function DrainPreview({ preview }: { preview: Previewed["preview"] }) {
  const plan = preview.data;
  const stillDownloading = plan === undefined ? undefined : downloadingWords(plan);
  return (
    <>
      {preview.isFetching && <Spinner label="Checking what this Move needs" />}
      {preview.isError && (
        <SignalLine tone="fail" announce>
          {sentenceOf(preview.error)}
        </SignalLine>
      )}
      {plan !== undefined && <Plan plan={plan} homed={homedWords(plan)} />}
      {stillDownloading !== undefined && <p>{stillDownloading}</p>}
    </>
  );
}
