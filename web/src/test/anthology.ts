import type {
  FilesCellList,
  SettingChoiceField,
  WorksContentType,
  WorksItem,
  WorksLevel,
  WorksWorkView,
} from "../api/gen/model";

/** A many-Item type with one level, an aside level, a code field and a per-Work option. */
export const anthologies: WorksContentType = {
  id: "anthology",
  name: "Anthologies",
  icon: "open-book",
  enabled: true,
  work: { one: "Anthology", many: "Anthologies" },
  levels: [{ one: "Volume", many: "Volumes" }],
  aside: { one: "Extra", many: "Extras" },
  item: { one: "Story", many: "Stories" },
  fields: [
    {
      key: "publisher",
      label: "Publisher",
      tier: "work",
      kind: "text",
      in: ["facts", "about"],
      list: { column: true, sort: true, filter: true },
      with: null,
      art: null,
    },
    {
      key: "volume",
      label: "Volume",
      tier: "level",
      depth: 1,
      kind: "number",
      in: ["details"],
      list: { column: false, sort: false, filter: false },
      with: null,
      art: null,
    },
    {
      key: "code",
      label: "Story",
      tier: "item",
      kind: "text",
      in: ["code"],
      list: { column: false, sort: false, filter: false },
      with: null,
      art: null,
    },
    {
      key: "published",
      label: "Published",
      tier: "item",
      kind: "date",
      in: ["about"],
      list: { column: false, sort: false, filter: false },
      with: null,
      art: null,
    },
  ],
  artwork: [
    { key: "cover", label: "Cover", tier: "work", shape: "poster" },
    { key: "spine", label: "Spine", tier: "level", depth: 1, shape: "poster" },
  ],
  source: "Fixture source",
  items: "many",
  art: { poster: "cover", backdrop: null, title: null },
  ratings: [],
  availability: "Next issue",
};

export const anthologyId = "6a8c0e2a-4c6e-4a8c-9e0a-2c4e6a8c0e2b";

/** Volumes, each of storiesPerVolume Stories, and one aside volume of Extras. */
export const volumeCount = 24;
export const storiesPerVolume = 49;
export const extraCount = 24;

const pad = (n: number) => String(n).padStart(2, "0");
const uuid = (kind: number, n: number) =>
  `0199f0b2-${kind.toString(16).padStart(4, "0")}-7000-8000-${n.toString(16).padStart(12, "0")}`;

function volume(number: number, aside: boolean): WorksLevel {
  return {
    id: uuid(1, number),
    parentId: null,
    depth: 1,
    key: `v${number}`,
    label: aside ? "Extras" : `Volume ${number}`,
    position: aside ? volumeCount + 1 : number,
    withdrawnAt: null,
    aside,
    fields: [{ key: "volume", kind: "number", number }],
    artwork: [],
  };
}

function story(level: WorksLevel, position: number, n: number): WorksItem {
  const code = level.aside ? `X${pad(position)}` : `V${pad(level.position)}S${pad(position)}`;
  return {
    id: uuid(2, n),
    levelId: level.id,
    key: `${level.key}s${position}`,
    label: `Story ${code}`,
    position,
    withdrawnAt: null,
    monitored: !level.aside,
    aside: level.aside,
    fields: [
      { key: "code", kind: "text", text: code },
      { key: "published", kind: "date", date: `2020-01-${pad((n % 28) + 1)}` },
    ],
    ratings: [],
    artwork: [],
    availability: { date: `2020-01-${pad((n % 28) + 1)}`, basis: "Published" },
  };
}

const order: SettingChoiceField = {
  key: "order",
  label: "Story order",
  kind: "choice",
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  default: "published",
  running: "published",
  value: "published",
  options: [
    { value: "published", label: "Published order" },
    { value: "reading", label: "Reading order" },
  ],
};

function tree(): Pick<WorksWorkView, "levels" | "items"> {
  const levels = [
    ...Array.from({ length: volumeCount }, (_, i) => volume(i + 1, false)),
    volume(volumeCount + 1, true),
  ];
  const items: WorksItem[] = [];
  for (const level of levels) {
    const stories = level.aside ? extraCount : storiesPerVolume;
    for (let position = 1; position <= stories; position++) {
      items.push(story(level, position, items.length + 1));
    }
  }
  return { levels, items };
}

/** The Long Shelf: 1,200 Stories over 24 Volumes and Extras, the Extras unmonitored. */
export const longShelf: WorksWorkView = {
  id: anthologyId,
  contentType: anthologies.id,
  key: "a1",
  label: "The Long Shelf",
  sortLabel: "Long Shelf",
  addedAt: "2026-10-12T09:00:00Z",
  refreshedAt: "2026-10-12T09:00:00Z",
  goneUpstreamAt: null,
  fields: [{ key: "publisher", kind: "text", text: "Harbor Press" }],
  ratings: [],
  artwork: [],
  ...tree(),
  links: [],
  profile: { id: "0199e1f2-7a3b-7c4d-8e5f-000000000001", name: "Films" },
  root: null,
  monitorNew: true,
  monitorAsides: false,
  levelFolders: true,
  options: [order],
};

/** Every Story of The Long Shelf without a cell. */
export const longShelfCells: FilesCellList = {
  cells: [],
  items: longShelf.items.map((item) => ({ itemId: item.id, status: null })),
  summary: { status: null, cells: {}, items: longShelf.items.length, itemStatus: {} },
  needsYou: [],
};
