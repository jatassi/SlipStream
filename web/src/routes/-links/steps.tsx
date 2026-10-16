import { type ComponentType, useState } from "react";

export type StepProps<F> = {
  readonly flow: F;
  /** Moves to the next step that shows, with what this step learned. */
  readonly next: (learned?: Partial<F>) => void;
};

/**
 * One row of a link's flow. Later specs add rows, so a flow is its table and
 * not a sequence of branches.
 */
export type Step<F> = {
  readonly id: string;
  readonly shows: (flow: F) => boolean;
  readonly Step: ComponentType<StepProps<F>>;
};

export function Steps<F>({ steps, start }: { steps: readonly Step<F>[]; start: F }) {
  const [{ flow, at }, setPlace] = useState({ flow: start, at: 0 });
  const next = (learned: Partial<F> = {}) => {
    const moved = { ...flow, ...learned };
    const following = steps.findIndex((step, index) => index > at && step.shows(moved));
    setPlace({ flow: moved, at: following === -1 ? at : following });
  };
  const current = steps[at];
  if (current === undefined) {
    throw new Error("a link's flow has no steps");
  }
  const shown = steps.filter((step) => step.shows(flow));
  return (
    <>
      <p className="text-meta text-ink-3">
        Step {shown.indexOf(current) + 1} of {shown.length}
      </p>
      <current.Step key={current.id} flow={flow} next={next} />
    </>
  );
}
