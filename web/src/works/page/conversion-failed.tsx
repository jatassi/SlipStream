import { Link } from "@tanstack/react-router";

import type { FilesConversionFailedNeed } from "../../api/gen/model";
import { SignalRow } from "../../ui/signal-row";
import { type Naming, conversionFailedSubject, conversionFailedVerdicts } from "./need-words";

/** A Conversion that failed for a cell of this Work, with the way to Conversions, where it is retried. */
export function ConversionFailed({
  naming,
  need,
}: {
  naming: Naming;
  need: FilesConversionFailedNeed;
}) {
  return (
    <SignalRow
      signal="fail"
      chip="Conversion failed"
      subject={conversionFailedSubject(naming, need)}
      verdict={conversionFailedVerdicts[need.class](naming.targetName(need.targetId))}
    >
      <Link
        to="/activity/conversions"
        search={{ band: "attention" }}
        className="text-fail-dim underline underline-offset-4"
      >
        Open Conversions
      </Link>
    </SignalRow>
  );
}
