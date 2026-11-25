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

export function StepCount({ number, of }: { number: number; of: number }) {
  return (
    <p className="text-meta text-ink-3">
      Step {number} of {of}
    </p>
  );
}

/** `after` counts the steps that follow on another page, as signup's on `/welcome`. */
export function Steps<F>({
  steps,
  start,
  after = 0,
}: {
  steps: readonly Step<F>[];
  start: F;
  after?: number;
}) {
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
      <StepCount number={shown.indexOf(current) + 1} of={shown.length + after} />
      <current.Step key={current.id} flow={flow} next={next} />
    </>
  );
}
