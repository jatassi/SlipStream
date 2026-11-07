import { Link } from "@tanstack/react-router";

import { Chevron } from "../ui/chevron";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { SectionDot } from "./section-alert";
import { settingsGroups } from "./sections";

/** The phone's Settings page: one list per group, each row a pushed section. */
export function SettingsList() {
  return settingsGroups.map((group) => (
    <Section key={group.id} title={group.label}>
      <List label={group.label}>
        {group.sections.map((section) => (
          <ListRow
            key={section.id}
            title={section.label}
            trailing={
              <span className="flex items-center gap-2">
                <SectionDot section={section} />
                <Chevron />
              </span>
            }
            render={<Link to="/settings/$section" params={{ section: section.id }} />}
          />
        ))}
      </List>
    </Section>
  ));
}
