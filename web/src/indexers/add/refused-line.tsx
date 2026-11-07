import { Alert02Icon } from "@hugeicons/core-free-icons";

import { Details } from "../../ui/details";
import { Icon } from "../../ui/icon";

/** A refusal, with the reason a definition doesn't load under Details. */
export type Refused = { readonly said: string; readonly reason: string | undefined };

export function RefusedLine({ refused }: { refused: Refused }) {
  const said = (
    <span role="alert" className="flex items-center gap-1.5 text-fail-ink">
      <Icon icon={Alert02Icon} className="size-4 shrink-0" />
      {refused.said}
    </span>
  );
  return refused.reason === undefined ? (
    said
  ) : (
    <Details summary={said}>
      <p className="mt-1 max-w-[64ch]">{refused.reason}</p>
    </Details>
  );
}
