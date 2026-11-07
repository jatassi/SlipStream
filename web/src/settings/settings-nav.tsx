import { Link } from "@tanstack/react-router";
import { useId } from "react";

import { SectionDot } from "./section-alert";
import type { SettingsGroup } from "./section";
import { settingsGroups } from "./sections";

/** The desktop column of Settings' sections, under their group labels. */
export function SettingsNav() {
  return (
    <nav aria-label="Settings" className="sticky top-20 flex w-50 shrink-0 flex-col gap-5">
      {settingsGroups.map((group) => (
        <NavGroup key={group.id} group={group} />
      ))}
    </nav>
  );
}

function NavGroup({ group }: { group: SettingsGroup }) {
  const labelId = useId();
  return (
    <div className="flex flex-col gap-1">
      <span id={labelId} className="px-2.5 text-meta font-medium text-ink-3">
        {group.label}
      </span>
      <ul aria-labelledby={labelId} className="flex flex-col gap-0.5">
        {group.sections.map((section) => (
          <li key={section.id}>
            <Link
              to="/settings/$section"
              params={{ section: section.id }}
              className="flex h-8 items-center gap-2.5 rounded-item px-2.5 text-body text-ink-2 transition-colors duration-(--duration-control) hover:bg-fill hover:text-ink-1 aria-[current=page]:bg-selected aria-[current=page]:font-medium aria-[current=page]:text-ink-1"
            >
              <span className="min-w-0 flex-1 truncate">{section.label}</span>
              <SectionDot section={section} />
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
