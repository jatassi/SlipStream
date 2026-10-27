import type { ReactNode } from "react";

import type { FilesState } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { LiveChip } from "../ui/live-chip";
import { MetPill } from "../ui/met-pill";
import { SignalChip } from "../ui/signal-chip";

/** Each cell state's chip, its word with its look, never colour alone. */
export const stateChips: { readonly [S in FilesState]: ReactNode } = {
  unreleased: <Chip look="outline">Unreleased</Chip>,
  missing: <Chip>Missing</Chip>,
  downloading: <LiveChip>Downloading</LiveChip>,
  failed: <SignalChip kind="fail">Failed</SignalChip>,
  upgradable: <Chip>Upgrade available</Chip>,
  met: <MetPill>Met</MetPill>,
};
