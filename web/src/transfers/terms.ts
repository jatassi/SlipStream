import {
  Alert02Icon,
  ArrowDataTransferHorizontalIcon,
  HardDriveIcon,
  HourglassIcon,
  Queue01Icon,
} from "@hugeicons/core-free-icons";

import type { TransfersTransferState } from "../api/gen/model";
import type { TermsFor, Tone } from "../ui/state-marks";

export const transferTerms = {
  queued: { word: "Queued", icon: Queue01Icon, tone: "quiet" },
  waiting: { word: "Waiting", icon: HourglassIcon, tone: "neutral" },
  needs_space: {
    word: "Needs space",
    icon: HardDriveIcon,
    tone: "warn",
    definition:
      "The files don't fit on the receiving Node, and the Transfer continues once there is space.",
  },
  copying: { word: "Transferring", icon: ArrowDataTransferHorizontalIcon, tone: "live" },
  failed: {
    word: "Failed",
    icon: Alert02Icon,
    tone: "fail",
    definition: "The Transfer stopped and doesn't resume.",
  },
} as const satisfies TermsFor<TransfersTransferState, Tone, "queued" | "waiting" | "copying">;
