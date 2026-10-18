import { useState } from "react";

import { isProblem } from "../api/errors";
import type { TargetsImpact } from "../api/gen/model";

export type Unwanted = "delete" | "keep";

type Changing = {
  readonly preview?: boolean;
  readonly unwanted?: Unwanted;
  readonly confirm?: number;
};

export type ImpactChange = {
  readonly title: string;
  /** What the save button says before its count, such as "Save" or "Delete Ultra". */
  readonly action: string;
  readonly run: (changing: Changing) => Promise<{ readonly impact: TargetsImpact }>;
  readonly done: () => Promise<unknown> | void;
};

const isNothing = (impact: TargetsImpact): boolean =>
  impact.works === 0 &&
  impact.ownList === 0 &&
  impact.unwanted.files === 0 &&
  impact.renamed === 0 &&
  impact.newlyWanted === 0 &&
  impact.downloading === 0;

export type Showing = {
  readonly change: ImpactChange;
  readonly impact: TargetsImpact;
  /** The save found more to delete than this preview's predecessor showed. */
  readonly changed: boolean;
};

export type ImpactSheetState = {
  readonly showing: Showing | undefined;
  readonly onSave: (showing: Showing, unwanted: Unwanted) => Promise<void>;
  readonly onClose: () => void;
};

/**
 * Runs a change's preview, then shows its Impact for the Admin to confirm,
 * or saves at once when the Impact shows nothing. A save that would delete
 * more than was confirmed is previewed again and shown with its fresh counts.
 */
export function useImpactSave(): {
  readonly save: (change: ImpactChange) => Promise<void>;
  readonly sheet: ImpactSheetState;
} {
  const [showing, setShowing] = useState<Showing>();

  const commit = async (change: ImpactChange, unwanted: Unwanted, confirmed: TargetsImpact) => {
    try {
      await change.run({ unwanted, confirm: confirmed.unwanted.files });
    } catch (error) {
      if (!isProblem(error, "impact_changed")) {
        throw error;
      }
      const { impact } = await change.run({ preview: true });
      setShowing({ change, impact, changed: true });
      return;
    }
    setShowing(undefined);
    await change.done();
  };

  const save = async (change: ImpactChange) => {
    const { impact } = await change.run({ preview: true });
    if (isNothing(impact)) {
      await commit(change, "delete", impact);
      return;
    }
    setShowing({ change, impact, changed: false });
  };

  return {
    save,
    sheet: {
      showing,
      onSave: (shown, unwanted) => commit(shown.change, unwanted, shown.impact),
      onClose: () => {
        setShowing(undefined);
      },
    },
  };
}
