import type { ComponentProps } from "react";

import type { FilesState } from "../api/gen/model";
import { statusMarks } from "./state-marks";

/** A cell's state as a 16px mark. Quiet dims it, for an Item nobody monitors. */
export function StateSquare({ state, quiet = false }: { state: FilesState; quiet?: boolean }) {
  return (
    <span
      className={`relative block size-4 shrink-0 rounded-[4px] ${statusMarks[state].square} ${quiet ? "opacity-55" : ""}`}
    />
  );
}

/**
 * A cell as a 42 × 30 button around its square. It is a plain button, so a
 * row can hold dozens. Its name is the label and the state's word.
 */
export function StateButton({
  state,
  label,
  quiet = false,
  ...props
}: Omit<ComponentProps<"button">, "children" | "className" | "aria-label" | "type"> & {
  state: FilesState;
  label: string;
  quiet?: boolean;
}) {
  return (
    <button
      {...props}
      type="button"
      aria-label={`${label}, ${statusMarks[state].word}`}
      className="inline-flex h-7.5 w-10.5 shrink-0 items-center justify-center rounded-item transition-[background-color] duration-(--duration-control) ease-standard hover:bg-[hsl(var(--h)_50%_70%/.1)] motion-reduce:transition-none"
    >
      <StateSquare state={state} quiet={quiet} />
    </button>
  );
}
