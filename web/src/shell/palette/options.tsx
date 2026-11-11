import { Add01Icon, Search01Icon } from "@hugeicons/core-free-icons";

import type {
  Bare,
  WorksCandidate,
  WorksContentType,
  WorksFieldSpec,
  WorksLibraryRow,
} from "../../api/gen/model";
import { statusMarks } from "../../ui/state-marks";
import type { PaletteGroup, PaletteOption } from "../../ui/command-palette";
import { Poster } from "../../ui/poster";
import { factTexts } from "../../works/page/facts";
import { unlistedLine } from "../../works/add/candidates";
import { openAddSheet } from "../../works/add/sheet-state";
import { addLabel, fieldText } from "../../works/fields";
import type { CommandContext } from "../commands";
// oxlint-disable-next-line import/max-dependencies -- an option leads with a poster and opens a Work, the add sheet or the add page
import { contentTypeIcons } from "../icons";
import { closePalette } from "../palette-state";
import type { CatalogResults } from "./provider";

export const closingThen = (run: () => void) => () => {
  closePalette();
  run();
};

function valueText(spec: WorksFieldSpec, value: Bare): string {
  if (Array.isArray(value)) {
    return value.join(", ");
  }
  return spec.kind === "duration" && typeof value === "number"
    ? fieldText({ key: spec.key, kind: "duration", ms: value })
    : String(value);
}

function firstFact(type: WorksContentType, work: WorksLibraryRow): string[] {
  const spec = type.fields.find((field) => field.in.includes("facts") && field.key in work.values);
  const value = spec === undefined ? undefined : work.values[spec.key];
  return spec === undefined || value === undefined ? [] : [valueText(spec, value)];
}

export function workOptions(
  works: readonly WorksLibraryRow[],
  contentTypes: readonly WorksContentType[],
  { navigate }: CommandContext,
): PaletteOption[] {
  const types = new Map(contentTypes.map((type) => [type.id, type]));
  return works.flatMap((work) => {
    const type = types.get(work.contentType);
    if (type === undefined) {
      return [];
    }
    const meta = [
      ...firstFact(type, work),
      type.work.one,
      ...(work.status === null ? [] : [statusMarks[work.status].word]),
    ];
    return [
      {
        id: work.id,
        label: work.label,
        meta: meta.join(" · "),
        icon: contentTypeIcons[type.icon],
        art: <Poster url={work.posterUrl ?? undefined} size="option" />,
        choose: closingThen(() => {
          void navigate({ to: "/works/$workId", params: { workId: work.id } });
        }),
      },
    ];
  });
}

function candidateOption(
  type: WorksContentType,
  candidate: WorksCandidate,
  { navigate }: CommandContext,
  seeAll: () => void,
): PaletteOption {
  const { workId } = candidate;
  const facts = factTexts(type, candidate.fields).slice(0, 1);
  const shown = {
    id: `${type.id}/${candidate.key}`,
    label: candidate.label,
    art: <Poster url={candidate.posterUrl ?? undefined} size="option" />,
  };
  if (workId !== null) {
    return {
      ...shown,
      meta: [...facts, "In the library"].join(" · "),
      icon: Add01Icon,
      choose: closingThen(() => {
        void navigate({ to: "/works/$workId", params: { workId } });
      }),
    };
  }
  if (candidate.unlisted) {
    return {
      ...shown,
      meta: [...facts, unlistedLine(type)].join(" · "),
      icon: Search01Icon,
      choose: seeAll,
    };
  }
  return {
    ...shown,
    meta: facts.join(" · "),
    icon: Add01Icon,
    choose: closingThen(() => {
      openAddSheet({ type, candidate });
    }),
  };
}

export function addGroup(
  { type, query, candidates }: CatalogResults,
  context: CommandContext,
): PaletteGroup {
  const seeAll = closingThen(() => {
    void context.navigate({ to: "/add", search: { contentType: type.id, query } });
  });
  return {
    label: addLabel(type),
    items: [
      ...candidates.map((candidate) => candidateOption(type, candidate, context, seeAll)),
      {
        id: `${type.id}/all`,
        label: "See all results",
        meta: type.work.many,
        icon: Search01Icon,
        choose: seeAll,
      },
    ],
  };
}
