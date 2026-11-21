import { useNavigate } from "@tanstack/react-router";

import { useGetSystemHealthSuspense } from "../api/gen/query/health/health";
import { itemAnchor, levels } from "../health/levels";
import { sectionHealth } from "../health/section-health";
import { Callout } from "../ui/callout";
import { AlertDot } from "../ui/count";
import type { SettingsSection } from "./section";

function useSectionHealth({ health }: SettingsSection) {
  const report = useGetSystemHealthSuspense().data;
  return health === undefined ? undefined : sectionHealth(report, health);
}

/** The dot on a section's row while a Health category it shows has a problem. */
export function SectionDot({ section }: { section: SettingsSection }) {
  const shown = useSectionHealth(section);
  return shown === undefined ? null : <AlertDot tone={shown.level} words={shown.words} />;
}

/** The section's worst Health item, at its top, leading to it on the Health page. */
export function SectionWarning({ section }: { section: SettingsSection }) {
  const shown = useSectionHealth(section);
  const navigate = useNavigate();
  if (shown === undefined) {
    return null;
  }
  const { category, item } = shown.first;
  return (
    <Callout
      tone={item.level}
      chip={levels[item.level].chip}
      subject={item.subject}
      since={item.since}
      action={{
        label: "Open Health",
        onClick: () => {
          void navigate({ to: "/health", hash: itemAnchor(category, item.key) });
        },
      }}
    >
      {item.text}
    </Callout>
  );
}
