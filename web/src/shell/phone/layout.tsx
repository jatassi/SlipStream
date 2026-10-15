import { type ReactNode, useState } from "react";

import { type ShownRow, tabRows, useAdminNav } from "../nav";
import { type Heading, ShellSlotsContext } from "../page";
import { useRouteTitle } from "../titles";
import { ConnectionLine } from "../connection-line";
import { useEdgeSwipeBack } from "./edge-swipe";
import { MoreSheet } from "./more-sheet";
import { NavBar } from "./nav-bar";
import { TabBar } from "./tab-bar";
import { useBack, usePhonePlace, useTabs } from "./tabs";

const barHeading = { kind: "bar" } as const;

export function PhoneShell({ children }: { children: ReactNode }) {
  const rows = useAdminNav();
  const tabs = tabRows(rows);
  const { current, open } = useTabs(tabs);
  const place = usePhonePlace();
  const back = useBack(current);
  const [titleOut, setTitleOut] = useState(false);
  useEdgeSwipeBack();

  const heading: Heading =
    place === "top" ? { kind: "large", onScrolledOut: setTitleOut } : barHeading;

  return (
    <ShellSlotsContext value={{ action: null, inspector: null, heading }}>
      <NavBar place={place} title={useRouteTitle()} back={back} titleShown={titleOut} />
      <main className="px-[max(16px,env(safe-area-inset-left))] pt-2 pb-[calc(env(safe-area-inset-bottom)+49px+48px)] [view-transition-name:content]">
        {children}
      </main>
      <div className="pointer-events-none fixed inset-x-0 bottom-[calc(env(safe-area-inset-bottom)+57px)] z-20 flex justify-center">
        <ConnectionLine />
      </div>
      <TabBar
        tabs={tabs}
        current={current}
        onOpen={(row: ShownRow) => {
          void open(row);
        }}
      />
      <MoreSheet rows={rows.filter((row) => row.tab === undefined)} />
    </ShellSlotsContext>
  );
}
