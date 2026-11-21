import { SignalLine } from "../../../../ui/signal-line";
import type { Dial } from "./state";
import { Waiting } from "./waiting";

const where = (addresses: readonly string[]) => addresses.join(" or ");

/** How the Controller's dial to a pasted Node address is going. */
export function DialLine({ dial, controller }: { dial: Dial; controller: string }) {
  if (dial.kind === "none") {
    return null;
  }
  if (dial.kind === "dialing") {
    return (
      <Waiting text={`${controller} is connecting to the new Node at ${where(dial.addresses)}…`} />
    );
  }
  return (
    <SignalLine tone="warn" announce>
      {`${controller} can't reach the new Node at ${where(dial.addresses)}. Check that the port is open.`}
    </SignalLine>
  );
}
