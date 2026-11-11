import type { ComponentProps } from "react";

import type { FilesState, FilesStateCounts } from "../api/gen/model";
import { stateOrder } from "../library/status";

export const stateWords: { readonly [S in FilesState]: string } = {
  failed: "Failed",
  upgradable: "Upgrade available",
  missing: "Missing",
  downloading: "Downloading",
  met: "Met",
  unreleased: "Unreleased",
};

const hatch =
  "bg-[repeating-linear-gradient(135deg,var(--color-ink-2)_0_1.5px,transparent_1.5px_4px)] ring-1 ring-hairline-strong ring-inset";
const unreleased = "ring-1 ring-hairline-strong ring-inset";

const squareLook: { readonly [S in FilesState]: string } = {
  failed: "bg-fail",
  upgradable:
    "ring-2 ring-solid ring-inset after:absolute after:inset-1.25 after:rounded-capsule after:bg-solid",
  missing: hatch,
  downloading:
    "overflow-hidden shadow-live-glow ring-1 ring-[hsl(var(--h)_90%_65%/.32)] ring-inset after:absolute after:inset-x-0 after:bottom-0 after:h-1/2 after:bg-linear-to-t after:from-glow after:to-solid",
  met: "bg-solid",
  unreleased,
};

const barLook: { readonly [S in FilesState]: string } = {
  failed: "bg-fail",
  upgradable: "ring-1 ring-solid ring-inset",
  missing: hatch,
  downloading: "bg-linear-to-r from-glow to-solid shadow-live-glow",
  met: "bg-solid",
  unreleased,
};

/** A cell's state as a 16px mark. Quiet dims it, for an Item nobody monitors. */
export function StateSquare({ state, quiet = false }: { state: FilesState; quiet?: boolean }) {
  return (
    <span
      className={`relative block size-4 shrink-0 rounded-[4px] ${squareLook[state]} ${quiet ? "opacity-55" : ""}`}
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
      aria-label={`${label}, ${stateWords[state]}`}
      className="inline-flex h-7.5 w-10.5 shrink-0 items-center justify-center rounded-item transition-[background-color] duration-(--duration-control) ease-standard hover:bg-[hsl(var(--h)_50%_70%/.1)] motion-reduce:transition-none"
    >
      <StateSquare state={state} quiet={quiet} />
    </button>
  );
}

/** A level's cells as one 180 × 8 bar, a segment per state in the one state order. */
export function StateBar({ counts }: { counts: FilesStateCounts }) {
  const shown = stateOrder.flatMap((state) => {
    const count = counts[state] ?? 0;
    return count > 0 ? [{ state, count }] : [];
  });
  const named = {
    role: "img",
    "aria-label":
      shown.length === 0
        ? "No cells"
        : shown.map(({ state, count }) => `${count} ${stateWords[state]}`).join(", "),
  } as const;
  return (
    <span
      {...named}
      className="flex h-2 w-45 shrink-0 gap-0.5 overflow-hidden rounded-[2px] bg-fill"
    >
      {shown.map(({ state, count }) => (
        <span
          key={state}
          style={{ flexGrow: count }}
          className={`basis-0 rounded-[2px] ${barLook[state]}`}
        />
      ))}
    </span>
  );
}
