import { Alert02Icon } from "@hugeicons/core-free-icons";

import { Icon } from "../../../../ui/icon";
import type { Dial } from "./state";
import { Waiting } from "./waiting";

const where = (addresses: readonly string[]) => addresses.join(" or ");

/** How the Controller's dial to a pasted Node address is going. */
export function DialLine({ dial, controller }: { dial: Dial; controller: string }) {
  if (dial.kind === "none") {
    return null;
  }
  if (dial.kind === "dialing") {
    return <Waiting text={`${controller} is dialing the new Node at ${where(dial.addresses)}…`} />;
  }
  return (
    <p role="alert" className="flex items-start gap-1.5 text-warn-ink">
      <Icon icon={Alert02Icon} className="mt-0.5 size-4 shrink-0" />
      {`${controller} can't reach the new Node at ${where(dial.addresses)}. Check that the port is open.`}
    </p>
  );
}
