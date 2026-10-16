import { Alert02Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

/** A failure said in one sentence, in fail-ink with its icon. */
export function AlertLine({ text }: { text: string }) {
  return (
    <p role="alert" className="flex items-center gap-1.5 text-fail-ink">
      <Icon icon={Alert02Icon} className="size-4 shrink-0" />
      {text}
    </p>
  );
}
