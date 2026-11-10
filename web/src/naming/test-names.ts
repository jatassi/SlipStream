import type { FormatsWorkRef, WorksContentType } from "../api/gen/model";
import { parseReleases } from "../api/gen/query/formats/formats";
import { previewNaming } from "../api/gen/query/naming/naming";
import { listLibrary } from "../api/gen/query/works/works";
import type { Draft } from "./card";
import type { Outcome, TestedFile } from "./tester-words";

type Matched = { readonly name: string; readonly at: number; readonly work: FormatsWorkRef };

/**
 * Reads each name as a library scan reads a found file, then names each one
 * that matches a Work under its Content Type's draft.
 */
export async function testNames(
  names: readonly string[],
  types: readonly WorksContentType[],
  draftOf: (type: string) => Draft,
): Promise<TestedFile[]> {
  const { results } = await parseReleases({ names: [...names], kind: "file" });
  const matched: Matched[] = results.flatMap(({ name, work }, at) =>
    work === null ? [] : [{ name, at, work }],
  );
  const workIds = [...new Set(matched.map(({ work }) => work.id))];
  const { rows } = workIds.length === 0 ? { rows: [] } : await listLibrary({ workIds });
  const typeOf = new Map(
    rows.map((row) => [row.id, types.find(({ id }) => id === row.contentType)]),
  );
  const byType = new Map<WorksContentType, Matched[]>();
  for (const one of matched) {
    const type = typeOf.get(one.work.id);
    if (type === undefined) {
      throw new Error(`the library lists no Content Type for the Work ${one.work.id}`);
    }
    byType.set(type, [...(byType.get(type) ?? []), one]);
  }
  const outcomes = new Map<number, Outcome>();
  await Promise.all(
    [...byType].map(async ([type, group]) => {
      const { tested } = await previewNaming(type.id, {
        draft: Object.fromEntries(draftOf(type.id)),
        names: group.map(({ name, work }) => ({ name, workId: work.id })),
      });
      for (const [index, { at, work }] of group.entries()) {
        const one = tested[index];
        outcomes.set(
          at,
          one === undefined
            ? { state: "refused", work, type }
            : { state: "tested", work, type, tested: one },
        );
      }
    }),
  );
  return results.map(({ name, facts, works }, at) => ({
    name,
    facts,
    outcome: outcomes.get(at) ?? { state: "unmatched", works },
  }));
}
