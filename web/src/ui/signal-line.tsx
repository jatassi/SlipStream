import type { ReactNode } from "react";

import { Icon } from "./icon";
import { type Signal, signals } from "./signal-chip";

export type SignalLineProps = {
  readonly tone: Signal;
  /**
   * The line reports what the person just did, so a screen reader reads it as
   * it appears: an alert for a failure, a status for a warning or a pass.
   */
  readonly announce?: true;
  /** One or two sentences. A link inside, such as the Movies Library, is fine. */
  readonly children: ReactNode;
};

type Look = { readonly icon: string; readonly text: string; readonly role: "alert" | "status" };

const looks: { readonly [S in Signal]: Look } = {
  fail: { icon: "text-fail", text: "text-fail-ink", role: "alert" },
  warn: { icon: "text-warn", text: "text-warn-ink", role: "status" },
  ok: { icon: "text-ok", text: "text-ink-1", role: "status" },
};

/** One sentence led by its signal's icon: the outcome of what was just done, beside the control. */
export function SignalLine({ tone, announce, children }: SignalLineProps) {
  const look = looks[tone];
  return (
    <p role={announce && look.role} className={`flex items-start gap-1.5 text-pretty ${look.text}`}>
      <Icon icon={signals[tone].icon} className={`mt-[0.2em] size-4 shrink-0 ${look.icon}`} />
      <span className="min-w-0">{children}</span>
    </p>
  );
}
