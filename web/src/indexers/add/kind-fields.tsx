import type { NodesNode } from "../../api/gen/model";
import { CatalogFields, catalogLead } from "./catalog-fields";
import { RssFields, TorznabFields } from "./feed-fields";
import type { Source } from "./source";
import type { AddForm } from "./use-add";

const leads = {
  torznab:
    "A Torznab feed reaches one Prowlarr or Jackett indexer. SlipStream tests the feed first and reads which Trackers its results come from.",
  rss: "An RSS feed lists a Tracker's newest items and is never searched. Saving checks the feed first.",
} as const;

/** What adding from source does, as the form's first sentence. */
export const leadOf = (source: Source, home: string): string =>
  source.kind === "catalog" ? catalogLead(source.form, home) : leads[source.kind];

/** The fields source's kind asks for. */
export function Fields({
  form,
  source,
  home,
}: {
  form: AddForm;
  source: Source;
  home: NodesNode | undefined;
}) {
  switch (source.kind) {
    case "catalog":
      return <CatalogFields form={form} definition={source.form} home={home} />;
    case "torznab":
      return <TorznabFields form={form} />;
    case "rss":
      return <RssFields form={form} contentTypes={source.contentTypes} />;
    default:
      return source satisfies never;
  }
}
