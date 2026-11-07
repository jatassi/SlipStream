import { ArrowRight01Icon, CodeIcon, RssIcon, Search01Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { Link, useNavigate } from "@tanstack/react-router";

import type { IndexersIndexerCatalog } from "../../api/gen/model";
import { Icon } from "../../ui/icon";
import { List, ListRow } from "../../ui/list";
import { Section } from "../../ui/section";
import { CatalogPicker } from "./catalog-picker";

type Way = {
  to:
    | "/settings/indexers/new/torznab"
    | "/settings/indexers/new/rss"
    | "/settings/indexers/new/custom";
  icon: IconSvgElement;
  title: string;
  meta: string;
};

const ways: readonly Way[] = [
  {
    to: "/settings/indexers/new/torznab",
    icon: Search01Icon,
    title: "Torznab feed",
    meta: "Reaches one Prowlarr or Jackett indexer by its feed address and API key.",
  },
  {
    to: "/settings/indexers/new/rss",
    icon: RssIcon,
    title: "RSS feed",
    meta: "Reaches one Tracker by its feed address. It lists new items and is never searched.",
  },
  {
    to: "/settings/indexers/new/custom",
    icon: CodeIcon,
    title: "Custom definition",
    meta: "Runs a Cardigann definition the catalog lacks, pasted or uploaded as YAML.",
  },
];

/** The first step of Add Indexer: a definition from the catalog, or a feed or custom definition. */
export function SourceStep({ catalog }: { catalog: IndexersIndexerCatalog }) {
  const navigate = useNavigate();
  return (
    <div className="flex max-w-[760px] flex-col">
      <p className="mb-6 text-ink-1">
        Search the Indexer catalog for the Tracker you use. For one the catalog lacks, add its
        Torznab or RSS feed, or a custom definition.
      </p>
      <Section title="Indexer catalog" count={catalog.entries.length}>
        <CatalogPicker
          catalog={catalog}
          onPick={(entry) => {
            void navigate({
              to: "/settings/indexers/new/catalog/$definitionId",
              params: { definitionId: entry.id },
            });
          }}
        />
      </Section>
      <Section title="Feeds and custom definitions">
        <List label="Feeds and custom definitions">
          {ways.map((way) => (
            <ListRow
              key={way.to}
              title={way.title}
              meta={way.meta}
              leading={<Icon icon={way.icon} className="size-4.5 shrink-0 text-ink-2" />}
              trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
              render={<Link to={way.to} />}
            />
          ))}
        </List>
      </Section>
    </div>
  );
}
