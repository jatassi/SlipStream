import type {
  IndexersDefinitionField,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersPortFacts,
  IndexersPortPin,
} from "../api/gen/model";
import type { PortForm } from "../indexers/port-form";
import { catalog, entry, notDownloaded } from "./catalog";
import { indexer } from "./indexers";
import { homeId } from "./nodes";

function field(
  over: Partial<IndexersDefinitionField> & Pick<IndexersDefinitionField, "name" | "type" | "label">,
) {
  return {
    default: null,
    help: null,
    options: [],
    min: null,
    max: null,
    sealed: false,
    required: false,
    ...over,
  } satisfies IndexersDefinitionField;
}

const port = (over: Parameters<typeof entry>[0]) => entry({ origin: "native", ...over });

export const hdbitsEntry = port({ id: "hdbits", name: "HDBits", categories: [2000, 5000] });
export const torrentsCsvEntry = port({
  id: "torrentscsv",
  name: "TorrentsCSV",
  privacy: "public",
  categories: [2000],
});
export const norbitsEntry = port({
  id: "norbits",
  name: "NorBits",
  language: "nb-NO",
  categories: [2000, 5000],
});
export const iptorrentsEntry = port({ id: "iptorrents", name: "IPTorrents" });
export const torrentDayEntry = port({ id: "torrentday", name: "TorrentDay" });
export const shazbatEntry = port({ id: "shazbat", name: "Shazbat", categories: [5000] });

const username = field({ name: "username", type: "text", label: "Username", required: true });
const password = field({
  name: "password",
  type: "password",
  label: "Password",
  sealed: true,
  required: true,
});

const unlimited = { count: 0, per: "hour" } as const;

const facts = (over: Partial<IndexersPortFacts> = {}): IndexersPortFacts => ({
  userAgent: false,
  searches: unlimited,
  grabs: unlimited,
  recent: true,
  spacingSeconds: 2,
  synced: "2026-11-06T15:30:00Z",
  ...over,
});

const portForm = (over: Partial<PortForm> & Pick<PortForm, "entry" | "links">): PortForm => ({
  session: false,
  captcha: false,
  solver: false,
  cookie: false,
  code: false,
  fields: [],
  port: facts(),
  ...over,
});

export const hdbitsForm = portForm({
  entry: hdbitsEntry,
  links: ["https://hdbits.org/"],
  fields: [
    username,
    field({ name: "passkey", type: "text", label: "Passkey", sealed: true, required: true }),
    field({
      name: "codecs",
      type: "multiselect",
      label: "Codecs",
      default: "1,5",
      options: [
        { value: "1", label: "H.264" },
        { value: "5", label: "HEVC" },
        { value: "3", label: "VC-1" },
      ],
    }),
    field({ name: "freeleechOnly", type: "checkbox", label: "Only freeleech", default: "false" }),
  ],
});

export const torrentsCsvForm = portForm({
  entry: torrentsCsvEntry,
  links: ["https://torrents-csv.com/"],
  port: facts({ recent: false }),
});

export const norbitsForm = portForm({
  entry: norbitsEntry,
  links: ["https://norbits.net/", "https://norbits.no/"],
  session: true,
  code: true,
  fields: [
    username,
    password,
    field({
      name: "authenticatorKey",
      type: "password",
      label: "Authenticator key",
      sealed: true,
      help: "NorBits shows this key when you turn on two-factor sign-in, usually under the QR code. With it, SlipStream makes each code and signs in again automatically. Leave it empty to type a code whenever a sign-in needs one.",
    }),
  ],
});

export const iptorrentsForm = portForm({
  entry: iptorrentsEntry,
  links: ["https://iptorrents.com/", "https://iptorrents.me/", "https://nemo.iptorrents.com/"],
  session: true,
  cookie: true,
  port: facts({ userAgent: true, spacingSeconds: 6.1 }),
});

export const shazbatForm = portForm({
  entry: shazbatEntry,
  links: ["https://www.shazbat.tv/"],
  fields: [
    field({ name: "passkey", type: "text", label: "Passkey", sealed: true, required: true }),
    field({
      name: "showPageLimit",
      type: "number",
      label: "Show page limit",
      default: "2",
      min: 1,
      max: 5,
    }),
  ],
});

/** A cookie site that takes no User-Agent, with Prowlarr's default limits. */
export const torrentDayForm = portForm({
  entry: torrentDayEntry,
  links: ["https://tday.love/", "https://torrentday.cool/"],
  session: true,
  cookie: true,
  port: facts({ searches: { count: 100, per: "day" }, grabs: { count: 20, per: "day" } }),
});

export const portForms: readonly PortForm[] = [
  hdbitsForm,
  torrentsCsvForm,
  norbitsForm,
  iptorrentsForm,
  shazbatForm,
  torrentDayForm,
];

const ports = portForms.map((form) => form.entry);

/** The catalog with the ported sites listed among its definitions. */
export const catalogWithPorts: IndexersIndexerCatalog = {
  ...catalog,
  entries: [...catalog.entries, ...ports],
};

/** A fresh install with no network: the ported sites, under why the catalog lists nothing else. */
export const portsBeforeCatalog: IndexersIndexerCatalog = { ...notDownloaded, entries: ports };

export const norbitsId = "9e4b3f7d-5c6a-4b8e-9d0f-2a3b4c5d6e7f";
export const iptorrentsId = "0f5c4a8e-6d7b-4c9f-8e1a-3b4c5d6e7f8a";

const synced = "2026-11-06T15:30:00Z";

const norbitsPort: IndexersPortPin = {
  id: "norbits",
  name: "NorBits",
  synced,
  baseAddress: "https://norbits.net/",
  values: { username: "ada" },
  sealed: ["password"],
};

/** A NorBits Indexer stopped because its sign-in needs a two-factor code. */
export const norbits: IndexersIndexer = indexer({
  id: norbitsId,
  name: "NorBits",
  kind: "native",
  signsIn: true,
  feedHost: null,
  port: norbitsPort,
  status: {
    state: "stopped",
    cause: "code_needed",
    text: null,
    until: null,
    since: "2026-11-07T07:00:00Z",
    nodeId: homeId,
    solverTried: false,
    rung: 0,
    lastFailure: null,
    successAt: "2026-11-06T09:00:00Z",
  },
  waits: {
    kind: "stopped",
    releases: 0,
    until: null,
    since: null,
    nodeId: null,
    lapsed: false,
    cause: "code_needed",
    text: null,
  },
  sentence: "Stopped because NorBits needs a two-factor code to sign in again.",
  fix: "Type the current code and test again.",
});

const iptorrentsPort: IndexersPortPin = {
  id: "iptorrents",
  name: "IPTorrents",
  synced,
  baseAddress: "https://iptorrents.example/",
  values: {},
  sealed: [],
};

/** An IPTorrents Indexer on a typed address, since IPTorrents moved off its known ones. */
export const iptorrents: IndexersIndexer = indexer({
  id: iptorrentsId,
  name: "IPTorrents",
  kind: "native",
  signsIn: true,
  feedHost: null,
  port: iptorrentsPort,
});

/** The port a new native Indexer saves, as the server returns it. */
export const portPinOf = (form: PortForm, baseAddress: string): IndexersPortPin => ({
  id: form.entry.id,
  name: form.entry.name,
  synced,
  baseAddress,
  values: {},
  sealed: [],
});
