import type {
  IndexersCaptchaImage,
  IndexersCatalogEntry,
  IndexersDefinitionField,
  IndexersDefinitionForm,
  IndexersIndexerCatalog,
} from "../api/gen/model";

export function entry(
  over: Partial<IndexersCatalogEntry> & Pick<IndexersCatalogEntry, "id" | "name">,
) {
  return {
    description: `${over.name} is a fictional Tracker.`,
    language: "en-US",
    privacy: "private",
    categories: [2000, 2040],
    origin: "catalog",
    reason: null,
    replacedByCustom: false,
    version: "2026-10-26T03:00:00Z",
    ...over,
  } satisfies IndexersCatalogEntry;
}

export const larkspurEntry = entry({ id: "larkspur", name: "Larkspur" });
export const quarryEntry = entry({
  id: "quarry",
  name: "Quarry",
  privacy: "semi-private",
  categories: [2000, 5000, 5040],
});
export const ferryEntry = entry({
  id: "ferry",
  name: "Ferry",
  privacy: "public",
  language: "fr-FR",
  categories: [2000],
});
export const cinderEntry = entry({ id: "cinder", name: "Cinder", privacy: "public" });
export const brokenEntry = entry({
  id: "broken",
  name: "Bramble",
  reason: "The definition signs in with the method 'oauth', which SlipStream doesn't run.",
});
export const marshEntry = entry({ id: "marsh", name: "Marsh", replacedByCustom: true });
export const customMarshEntry = entry({ id: "marsh", name: "Marsh", origin: "custom" });
export const meadowEntry = entry({ id: "meadow", name: "Meadow", categories: [2000, 5000, 7000] });

export const catalog: IndexersIndexerCatalog = {
  address: "https://indexers.prowlarr.com/master",
  addressIsDefault: true,
  refreshedAt: "2026-10-27T03:00:00Z",
  nextRefreshAt: "2026-10-28T03:00:00Z",
  notice: null,
  entries: [
    brokenEntry,
    cinderEntry,
    ferryEntry,
    larkspurEntry,
    marshEntry,
    meadowEntry,
    quarryEntry,
    customMarshEntry,
  ],
};

export const notDownloaded: IndexersIndexerCatalog = {
  ...catalog,
  refreshedAt: null,
  entries: [],
  notice: "SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at 03:00.",
};

function field(
  over: Partial<IndexersDefinitionField> & Pick<IndexersDefinitionField, "name" | "type">,
) {
  return {
    label: "",
    default: null,
    help: null,
    options: [],
    min: null,
    max: null,
    sealed: false,
    ...over,
  } satisfies IndexersDefinitionField;
}

const username = field({ name: "username", type: "text", label: "Username" });
const password = field({ name: "password", type: "password", label: "Password", sealed: true });
const captchaField = field({
  name: "cardigannCaptcha",
  type: "cardigannCaptcha",
  label: "CAPTCHA",
});

export const larkspurForm: IndexersDefinitionForm = {
  entry: larkspurEntry,
  links: ["https://larkspur.example/"],
  session: true,
  captcha: true,
  solver: false,
  cookie: false,
  code: false,
  port: null,
  fields: [
    username,
    password,
    field({
      name: "sort",
      type: "select",
      label: "Sort requested from site",
      default: "added",
      options: [
        { value: "added", label: "created" },
        { value: "seeders", label: "seeders" },
      ],
    }),
    captchaField,
  ],
};

export const quarryForm: IndexersDefinitionForm = {
  entry: quarryEntry,
  links: ["https://quarry.example/", "https://quarry.example.net/"],
  session: false,
  captcha: false,
  solver: false,
  cookie: false,
  code: false,
  port: null,
  fields: [
    field({ name: "apikey", type: "text", label: "API key", sealed: true }),
    field({
      name: "info_key",
      type: "info",
      label: "About the API key",
      help: "Your API key is on your <b>Quarry</b> profile page.",
    }),
  ],
};

export const cinderForm: IndexersDefinitionForm = {
  entry: cinderEntry,
  links: ["https://cinder.example/"],
  session: true,
  captcha: false,
  solver: true,
  cookie: true,
  code: false,
  port: null,
  fields: [
    field({ name: "info_cookie", type: "info_cookie" }),
    field({ name: "info_flaresolverr", type: "info_flaresolverr" }),
  ],
};

/** A definition with every Cardigann field type, in the order the form draws them. */
export const meadowForm: IndexersDefinitionForm = {
  entry: meadowEntry,
  links: ["https://meadow.example/"],
  session: true,
  captcha: true,
  solver: true,
  cookie: false,
  code: false,
  port: null,
  fields: [
    username,
    password,
    field({ name: "2fa-code", type: "text", label: "Two-factor code", sealed: true }),
    field({ name: "freeleech", type: "checkbox", label: "Only freeleech", default: "false" }),
    field({
      name: "sort",
      type: "select",
      label: "Sort by",
      default: "seeders",
      options: [
        { value: "added", label: "Added" },
        { value: "seeders", label: "Seeders" },
      ],
    }),
    field({ name: "info", type: "info", label: "Ratio", help: "Keep a ratio of 0.8 or more." }),
    field({ name: "info_cookie", type: "info_cookie", help: "Copy the cookie from your browser." }),
    field({
      name: "info_useragent",
      type: "info_useragent",
      help: "Copy your browser's User-Agent.",
    }),
    field({
      name: "info_8000",
      type: "info_category_8000",
      help: "Category 8000 holds everything the site doesn't sort.",
    }),
    field({ name: "info_flaresolverr", type: "info_flaresolverr", help: "Use FlareSolverr." }),
    captchaField,
  ],
};

export const brokenForm: IndexersDefinitionForm = {
  entry: brokenEntry,
  links: [],
  session: false,
  captcha: false,
  solver: false,
  cookie: false,
  code: false,
  port: null,
  fields: [],
};

export const forms: readonly IndexersDefinitionForm[] = [
  larkspurForm,
  quarryForm,
  cinderForm,
  meadowForm,
  brokenForm,
];

const pixel = (body: string) => `data:image/png;base64,${body}`;

/** Captcha images in the order the server hands them out. */
export const captchas: readonly IndexersCaptchaImage[] = [
  {
    attempt: "attempt-1",
    image: pixel(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
    ),
    expiresAt: "2026-10-27T12:05:00Z",
  },
  {
    attempt: "attempt-2",
    image: pixel(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
    ),
    expiresAt: "2026-10-27T12:10:00Z",
  },
  {
    attempt: "attempt-3",
    image: pixel(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
    ),
    expiresAt: "2026-10-27T12:15:00Z",
  },
];
