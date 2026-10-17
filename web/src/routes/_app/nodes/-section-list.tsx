import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../../../api/gen/model";
import { nodeSections } from "../../../shell/node-sections";
import { Chevron } from "../../../ui/chevron";
import { Icon } from "../../../ui/icon";
import { List, ListRow } from "../../../ui/list";
import { Section } from "../../../ui/section";

/** On phone, the Node's page lists its other sections, each a pushed page. */
export function SectionList({ node }: { node: NodesNode }) {
  return (
    <Section title="Sections">
      <List label={node.name}>
        {nodeSections
          .filter((section) => section.id !== "overview")
          .map((section) => (
            <ListRow
              key={section.id}
              title={section.label}
              leading={<Icon icon={section.icon} className="size-5 shrink-0 text-ink-2" />}
              trailing={<Chevron />}
              render={<Link to={section.to} params={{ nodeId: node.id }} />}
            />
          ))}
      </List>
    </Section>
  );
}
