import { Add01Icon, Search01Icon } from "@hugeicons/core-free-icons";

import type {
  Bare,
  FilesState,
  WorksContentType,
  WorksFieldSpec,
  WorksLibraryRow,
} from "../../api/gen/model";
import type { PaletteGroup, PaletteOption } from "../../ui/command-palette";
import { Poster } from "../../ui/poster";
import { factTexts } from "../../works/page/facts";
import { openAddSheet } from "../../works/add/sheet-state";
import { addLabel, fieldText } from "../../works/fields";
import type { CommandContext } from "../commands";
import { contentTypeIcons } from "../icons";
import { closePalette } from "../palette-state";
// oxlint-disable-next-line import/max-dependencies -- an option leads with a poster and opens a Work, the add sheet or the add page
import type { CatalogResults } from "./provider";

export const closingThen = (run: () => void) => () => {
  closePalette();
  run();
};

const statusWords: { readonly [S in FilesState]: string } = {
  unreleased: "Unreleased",
  missing: "Missing",
  downloading: "Downloading",
  failed: "Failed",
  upgradable: "Upgradable",
  met: "Met",
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
      ...(work.status === null ? [] : [statusWords[work.status]]),
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

export function addGroup(
  { type, query, candidates }: CatalogResults,
  { navigate }: CommandContext,
): PaletteGroup {
  const options: PaletteOption[] = candidates.map((candidate) => {
    const { workId } = candidate;
    const meta = [
      ...factTexts(type, candidate.fields).slice(0, 1),
      ...(workId === null ? [] : ["In the library"]),
    ];
    return {
      id: `${type.id}/${candidate.key}`,
      label: candidate.label,
      meta: meta.join(" · "),
      icon: Add01Icon,
      art: <Poster url={candidate.posterUrl ?? undefined} size="option" />,
      choose: closingThen(() => {
        if (workId === null) {
          openAddSheet({ type, candidate });
          return;
        }
        void navigate({ to: "/works/$workId", params: { workId } });
      }),
    };
  });
  return {
    label: addLabel(type),
    items: [
      ...options,
      {
        id: `${type.id}/all`,
        label: "See all results",
        meta: type.work.many,
        icon: Search01Icon,
        choose: closingThen(() => {
          void navigate({ to: "/add", search: { contentType: type.id, query } });
        }),
      },
    ],
  };
}
