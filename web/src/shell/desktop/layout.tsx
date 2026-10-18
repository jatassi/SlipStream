import { type ReactNode, useState } from "react";

import { ShellSlotsContext } from "../page";
import { Sidebar } from "./sidebar";
import { ActionSlotTarget, TopBar } from "./top-bar";

const contentHeading = { kind: "content" } as const;

export function DesktopShell({ children }: { children: ReactNode }) {
  const [action, setAction] = useState<HTMLElement | null>(null);
  const [inspector, setInspector] = useState<HTMLElement | null>(null);
  return (
    <ShellSlotsContext value={{ action, inspector, heading: contentHeading }}>
      <div className="grid min-h-dvh grid-cols-[auto_minmax(0,1fr)]">
        <Sidebar />
        <div className="flex min-w-0 flex-col">
          <ActionSlotTarget value={setAction}>
            <TopBar />
          </ActionSlotTarget>
          <div className="grid flex-1 grid-cols-[minmax(0,1fr)_auto]">
            <main className="mx-auto w-full max-w-[1480px] px-7 pt-6 pb-30 [view-transition-name:content]">
              {children}
            </main>
            <div
              ref={setInspector}
              className="sticky top-14 h-[calc(100dvh-3.5rem)] py-3 pr-3 empty:hidden"
            />
          </div>
        </div>
      </div>
    </ShellSlotsContext>
  );
}
