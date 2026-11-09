import { type ReactElement, useState } from "react";

import type { WorksContentType } from "../api/gen/model";
import { planRenames } from "../api/gen/query/files/files";
import { type Opening, PlanSheet } from "../renames/plan-sheet";

type Values = Readonly<Record<string, unknown>>;

/**
 * A card's way to its type's rename plan: on demand, and before a save
 * that renames files, where the plan offers Save and rename or Save only.
 */
export function useRenamePlan(
  type: WorksContentType,
  store: (values: Values) => Promise<void>,
): {
  /** Stores values at once when they rename no file, and otherwise shows their plan first. A refused draft throws. */
  readonly save: (values: Values) => Promise<void>;
  readonly review: () => void;
  readonly sheet: ReactElement;
} {
  const [opening, setOpening] = useState<Opening>();
  const noun = type.work.one.toLowerCase();
  const scope = { kind: "type", id: type.id } as const;
  return {
    save: async (values) => {
      const draft = { contentType: type.id, values };
      const plan = await planRenames({ scope, draft });
      if (plan.files === 0) {
        await store(values);
        return;
      }
      setOpening({
        title: `Save ${noun} naming`,
        scope,
        saving: { draft, plan, store: () => store(values) },
      });
    },
    review: () => {
      setOpening({ title: `Rename ${noun} files`, scope });
    },
    sheet: (
      <PlanSheet
        opening={opening}
        onClose={() => {
          setOpening(undefined);
        }}
      />
    ),
  };
}
