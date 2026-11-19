import { Fragment, type ReactNode } from "react";

/** Parts of a line of text with an ink-4 " · " between them, leaving out each part that's false, undefined or "". */
export function Joined({ parts }: { parts: readonly ReactNode[] }) {
  return parts
    .filter((part) => part !== undefined && part !== false && part !== "")
    .map((part, at) => (
      // oxlint-disable-next-line react/no-array-index-key -- the parts are fixed in place for each line
      <Fragment key={at}>
        {at > 0 && <span className="text-ink-4"> · </span>}
        {part}
      </Fragment>
    ));
}
