import type { WorksContentType, WorksTree } from "../../api/gen/model";
import { countWords } from "../../format";
import { useAmbient } from "../../shell/ambient/use-ambient";
import { Artwork, sized } from "../../ui/artwork";
import { factTexts } from "../page/facts";
import { artOf } from "../page/hero-art";
import { lowered } from "../monitoring/labels";

const neutral = { kind: "neutral" } as const;

/** The live levels and Items of the main run, asides left out, such as "3 seasons · 28 episodes". */
function sizeTexts(type: WorksContentType, work: WorksTree): string[] {
  const [outermost] = type.levels;
  if (type.items === "one" || outermost === undefined) {
    return [];
  }
  const levels = work.levels.filter(
    (level) => level.depth === 1 && !level.aside && level.withdrawnAt === null,
  );
  const items = work.items.filter((item) => !item.aside && item.withdrawnAt === null);
  return [
    countWords(levels.length, lowered(outermost)),
    countWords(items.length, lowered(type.item)),
  ];
}

export function Preview({ type, work }: { type: WorksContentType; work: WorksTree }) {
  const backdrop = artOf(work, type.art.backdrop);
  useAmbient(
    backdrop === undefined
      ? neutral
      : { kind: "artwork", url: sized(backdrop, "large"), workId: work.id },
  );
  const overviewKey = type.fields.find((spec) => spec.in.includes("overview"))?.key;
  const overview = work.fields.find((field) => field.key === overviewKey);
  return (
    <div className="flex gap-4">
      <Artwork use="poster" size="preview" url={artOf(work, type.art.poster)} label={work.label} />
      <div className="flex min-w-0 flex-col gap-2">
        <p className="text-meta text-ink-2">
          {[...factTexts(type, work.fields), ...sizeTexts(type, work)].join(" · ")}
        </p>
        {overview?.kind === "text" && (
          <p className="line-clamp-5 text-body text-ink-2">{overview.text}</p>
        )}
      </div>
    </div>
  );
}
