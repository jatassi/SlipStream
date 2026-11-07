import type { ReactNode } from "react";

import "./rows.css";

/**
 * One set of column tracks for every Table inside, sized by the content of
 * all of them. The first column takes the room the others leave. Pass the
 * columns the tables use, so the tracks follow them.
 */
export function SharedColumns({
  columns: [, ...rest],
  children,
}: {
  columns: readonly [unknown, ...unknown[]];
  children: ReactNode;
}) {
  const tracks = ["minmax(10.5rem, 1fr)", ...rest.map(() => "auto")].join(" ");
  return (
    <div className="@container [&+section]:mt-6">
      <div data-shared-columns="" style={{ gridTemplateColumns: tracks }}>
        {children}
      </div>
    </div>
  );
}
