import { HttpResponse, http } from "msw";

import type {
  NamingFieldPreview,
  NamingFieldTokens,
  NamingProblem,
  NamingToken,
  NamingTypeNaming,
  SettingField,
  SettingTemplateField,
} from "../api/gen/model";
import { PreviewNamingBody, SetNamingBody } from "../api/gen/zod/naming/naming.zod";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const template = (key: string, label: string, text: string): SettingTemplateField => ({
  ...about,
  kind: "template",
  key,
  label,
  isDefault: true,
  value: text,
  default: text,
  field: key.slice(key.indexOf(".naming.") + ".naming.".length),
  plain: false,
});

const colonSettings = (key: (field: string) => string): SettingField[] => [
  {
    ...about,
    kind: "choice",
    key: key("colon"),
    label: "Colon replacement",
    isDefault: true,
    value: "smart",
    running: "smart",
    default: "smart",
    options: [
      { value: "delete", label: "Delete" },
      { value: "smart", label: "Smart" },
      { value: "custom", label: "Custom" },
    ],
  },
  { ...template(key("colonCustom"), "Custom colon replacement", ""), plain: true },
];

const multiEpisodeSetting = (key: string): SettingField => ({
  ...about,
  kind: "choice",
  key,
  label: "Multi-story style",
  isDefault: true,
  value: "prefixed_range",
  running: "prefixed_range",
  default: "prefixed_range",
  options: [
    { value: "extend", label: "Extend" },
    { value: "prefixed_range", label: "Prefixed range" },
  ],
});

/** A type's naming settings as the server lists them, every one at its default. */
export function namingSettings(
  type: string,
  templates: readonly (readonly [field: string, label: string, text: string])[],
  multiEpisode = false,
): SettingField[] {
  const key = (field: string) => `${type}.naming.${field}`;
  return [
    {
      ...about,
      kind: "switch",
      key: key("renameFiles"),
      label: "Rename files",
      isDefault: true,
      value: true,
      running: true,
      default: true,
    },
    ...templates.map(([field, label, text]) => template(key(field), label, text)),
    ...colonSettings(key),
    ...(multiEpisode ? [multiEpisodeSetting(key("multiEpisode"))] : []),
  ];
}

const token = (name: string, part: Partial<NamingToken>): NamingToken => ({
  name,
  aliases: [],
  kind: "text",
  scope: "file",
  group: "release",
  example: name,
  about: `The ${name}.`,
  modifier: "cut",
  tags: [],
  ...part,
});

export const filmTokens: NamingToken[] = [
  token("Film Title", {
    kind: "title",
    scope: "work",
    group: "work",
    example: "The Quiet Orchard",
    about: "The film's English title.",
  }),
  token("Year", {
    kind: "year",
    scope: "work",
    group: "work",
    example: "2021",
    aliases: ["Release Year"],
    modifier: null,
    about: "The year the film came out.",
  }),
  token("ImdbId", {
    kind: "id",
    scope: "work",
    group: "work",
    example: "tt0113277",
    modifier: null,
    tags: ["imdb", "imdbid"],
    about: "The film's IMDb id.",
  }),
];

export const fileTokens: NamingToken[] = [
  ...filmTokens,
  token("Target", { kind: "id", group: "target", example: "4K HDR", modifier: null }),
  token("Edition", { group: "target", example: "Director's Cut", tags: ["edition"] }),
  token("Edition Tag", { group: "target", example: "{edition-Director's Cut}" }),
  token("Release Group", { example: "FLUX" }),
  token("MediaInfo AudioLanguages", {
    kind: "list",
    group: "media",
    example: "[EN+DE]",
    modifier: "languages",
  }),
];

/** One type's naming on a server that stores what a save sends and previews a draft. */
export type NamingServerType = {
  readonly naming: NamingTypeNaming;
  readonly tokens: readonly NamingFieldTokens[];
  /** The save check's sentences for a set of values by key. */
  readonly check?: (values: Readonly<Record<string, unknown>>) => NamingProblem[];
};

/** A naming setting with a value a save sent: null puts back its default. */
export function withValue(field: SettingField, value: unknown): SettingField {
  if (value === null) {
    return Object.assign(structuredClone(field), { value: field.default, isDefault: true });
  }
  const fits = field.kind === "switch" ? typeof value === "boolean" : typeof value === "string";
  return fits ? Object.assign(structuredClone(field), { value, isDefault: false }) : field;
}

function applied(
  settings: readonly SettingField[],
  values: Readonly<Record<string, unknown>>,
): SettingField[] {
  return settings.map((setting) =>
    setting.key in values ? withValue(setting, values[setting.key]) : setting,
  );
}

const valuesOf = (settings: readonly SettingField[]): Record<string, unknown> =>
  Object.fromEntries(settings.map((setting) => [setting.key, setting.value]));

/** A template's samples are its text as a path, so a preview shows what was typed. */
function previewOf(type: NamingServerType, draft: Readonly<Record<string, unknown>>) {
  const settings = applied(type.naming.settings, draft);
  const problems = type.check?.(valuesOf(settings)) ?? [];
  const fields: NamingFieldPreview[] = settings.map((setting) => ({
    key: setting.key,
    samples:
      setting.kind === "template" && !setting.plain
        ? [{ label: "The Quiet Orchard", path: `${setting.value}.mkv` }]
        : [],
    problems: problems.filter(({ key }) => key === setting.key).map(({ sentence }) => sentence),
  }));
  return { fields };
}

const refusal = (problems: readonly NamingProblem[]) =>
  HttpResponse.json(
    {
      status: 422,
      code: "naming_template_refused",
      title: "The naming settings don't pass the save check",
      errors: problems.map(({ key, sentence }) => ({
        location: `body.values.${key}`,
        message: sentence,
      })),
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

/** The naming endpoints over types, keeping what each save sends. */
export function namingServer(types: readonly NamingServerType[]) {
  const sent: Record<string, unknown>[] = [];
  const state = new Map(types.map((type) => [type.naming.contentType, type]));
  const typeOf = (id: string) => {
    const type = state.get(id);
    if (type === undefined) {
      throw new Error(`no naming for ${id}`);
    }
    return type;
  };
  const handlers = [
    http.get("*/api/v1/naming", () =>
      HttpResponse.json({ contentTypes: [...state.values()].map((type) => type.naming) }),
    ),
    http.get<{ type: string }>("*/api/v1/naming/:type/tokens", ({ params }) =>
      HttpResponse.json({ fields: typeOf(params.type).tokens }),
    ),
    http.post<{ type: string }>("*/api/v1/naming/:type/preview", async ({ params, request }) => {
      const { draft } = PreviewNamingBody.parse(await request.json());
      return HttpResponse.json(previewOf(typeOf(params.type), draft));
    }),
    http.patch<{ type: string }>("*/api/v1/naming/:type", async ({ params, request }) => {
      const type = typeOf(params.type);
      const { values } = SetNamingBody.parse(await request.json());
      sent.push(values);
      const settings = applied(type.naming.settings, values);
      const problems = type.check?.(valuesOf(settings)) ?? [];
      if (problems.length > 0) {
        return refusal(problems);
      }
      const naming = { ...type.naming, settings, problems: [] };
      state.set(naming.contentType, { ...type, naming });
      return HttpResponse.json(naming);
    }),
  ];
  return { handlers, sent };
}
