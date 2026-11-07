import { z } from "zod";

import type { IndexersDefinitionForm, IndexersIndexer, IndexersPortPin } from "../api/gen/model";

/*
 * Members the web reads that P0's API shape lacks and the Controller lane
 * adds, with indexer_test_failed's cause in captcha-of.ts. Each is parsed
 * here, so the rest of the web holds typed values, and this file goes once
 * web/src/api/gen carries them.
 */

const portFormMembers = z.object({
  /** The least time between two requests to the site, in seconds. */
  spacing: z.number().positive(),
  /** The site has a list of new Releases, which RSS sync reads. */
  recent: z.boolean(),
});

/** A ported site's form: its catalog-shaped form with its spacing and whether it lists new Releases. */
export type PortForm = IndexersDefinitionForm & z.output<typeof portFormMembers>;

export const portFormOf = (form: IndexersDefinitionForm): PortForm => ({
  ...form,
  ...portFormMembers.parse(form),
});

const portPinMembers = z.object({ name: z.string().min(1) });

/** A native Indexer's port, with the name of its site. */
export type PortPin = IndexersPortPin & z.output<typeof portPinMembers>;

export const portOf = (indexer: IndexersIndexer): PortPin | null =>
  indexer.port === null ? null : { ...indexer.port, ...portPinMembers.parse(indexer.port) };
