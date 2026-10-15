import { SidebarLeftIcon } from "@hugeicons/core-free-icons";
import { IconButton } from "../../ui/icon-button";
import { Mark } from "../../ui/mark";
import { BrandLink } from "../brand-link";
import { useAdminNav } from "../nav";
import { toggleSidebar, useSidebarCollapsed } from "../sidebar-state";
import { useMarkTransitionName } from "../startup/startup";
import { AccountFoot } from "./account-foot";
import { NavRows } from "./nav-rows";

export function Sidebar() {
  const collapsed = useSidebarCollapsed();
  const rows = useAdminNav();
  return (
    <div
      className={`sticky top-0 flex h-dvh flex-col bg-chrome shadow-[inset_-1px_0_0_var(--color-hairline)] transition-[width] duration-(--duration-inspector) ease-standard [view-transition-name:sidebar] motion-reduce:transition-none ${collapsed ? "w-15" : "w-57"}`}
    >
      <Brand collapsed={collapsed} />
      <nav
        aria-label="Main"
        className="flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto px-2.5 py-2"
      >
        <NavRows rows={rows.filter((row) => row.place === "main")} collapsed={collapsed} />
        <hr className="mx-2 my-3 h-px shrink-0 border-0 bg-hairline" />
        <NavRows rows={rows.filter((row) => row.place === "system")} collapsed={collapsed} />
      </nav>
      <AccountFoot collapsed={collapsed} />
    </div>
  );
}

function Brand({ collapsed }: { collapsed: boolean }) {
  const markName = useMarkTransitionName();
  return (
    <div
      className={`flex shrink-0 gap-2 px-3.5 pt-3 ${collapsed ? "flex-col items-center px-0" : "h-14 items-center justify-between pt-0"}`}
    >
      <BrandLink
        to="/library"
        aria-label="SlipStream"
        className="flex items-center gap-2 rounded-item text-[17px] tracking-[-0.03em] text-ink-1"
      >
        <Mark
          size={24}
          decorative
          {...(markName === undefined ? {} : { viewTransitionName: markName })}
        />
        {!collapsed && (
          <span>
            <span className="font-semibold">Slip</span>
            <span className="font-normal">Stream</span>
          </span>
        )}
      </BrandLink>
      <IconButton
        icon={SidebarLeftIcon}
        label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        aria-expanded={!collapsed}
        onClick={toggleSidebar}
      />
    </div>
  );
}
