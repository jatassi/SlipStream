import type {
  IndexersDefinitionForm,
  IndexersNewIndexer,
  NodesNode,
  WorksContentType,
} from "../../api/gen/model";
import { type LimitValues, noLimit } from "../fields/limit-field";
import { limitOf, limitSchema } from "../edit-values";
import type { PortForm } from "../port-shape";
import { type AddressValues, addressOf, startAddress } from "../fields/address-field";
import { settingAt, settingsOf, startValue } from "../fields/settings";
import { familiesOf } from "./families";

/** What a new Indexer is added from: a definition, a ported site, or a feed of a kind. */
export type Source =
  | { readonly kind: "catalog"; readonly form: IndexersDefinitionForm }
  | { readonly kind: "native"; readonly form: PortForm }
  | { readonly kind: "torznab" }
  | { readonly kind: "rss"; readonly contentTypes: readonly WorksContentType[] };

/**
 * The add form's values for every kind; each kind reads its own. A
 * definition's settings are by field index, since their names may hold "."
 * or "[".
 */
export type AddValues = {
  name: string;
  homeNodeId: string;
  priority: string;
  searches: LimitValues;
  grabs: LimitValues;
  on: boolean;
  automaticSearch: boolean;
  rss: boolean;
  baseAddress: AddressValues;
  values: string[];
  families: string[];
  cookie: string;
  userAgent: string;
  address: string;
  apiKey: string;
  trackerName: string;
  media: string[];
  allowNoSize: boolean;
  captcha: string;
  code: string;
};

/** The form of a source drawn from a definition's fields, a catalog one or a port's. */
export const formOf = (source: Source) =>
  source.kind === "catalog" || source.kind === "native" ? source.form : undefined;

/** A ported site with no list of new Releases keeps RSS off, since RSS sync can't read it. */
export const listsRecent = (source: Source) => source.kind !== "native" || source.form.recent;

export function startValues(source: Source, home: NodesNode | undefined, name = ""): AddValues {
  const form = formOf(source);
  const enabled = source.kind === "rss" ? source.contentTypes.filter((type) => type.enabled) : [];
  return {
    name: form?.entry.name ?? name,
    homeNodeId: home?.id ?? "",
    priority: "25",
    searches: noLimit,
    grabs: noLimit,
    on: true,
    automaticSearch: true,
    rss: listsRecent(source),
    baseAddress: startAddress(form?.links ?? []),
    values: form?.fields.map(startValue) ?? [],
    families: form === undefined ? [] : familiesOf(form.entry.categories).map(String),
    cookie: "",
    userAgent: "",
    address: "",
    apiKey: "",
    trackerName: "",
    media: enabled.length === 1 ? enabled.map((type) => type.id) : [],
    allowNoSize: false,
    captcha: "",
    code: "",
  };
}

/** The username a captcha is shown for, which a new one needs once it changes. */
export function signInOf(source: Source, values: AddValues): string {
  const form = formOf(source);
  if (form === undefined) {
    return "";
  }
  const username = form.fields.findIndex((field) => field.name === "username");
  return `${values.homeNodeId}\n${username < 0 ? "" : values.values[username]}`;
}

/** The feed a Torznab Test read, which a new Test needs once it changes. */
export const feedOf = (values: AddValues) => `${values.address.trim()}\n${values.apiKey}`;

/** The categories a narrowed choice of families keeps; null for all of them. */
function categoriesOf(form: IndexersDefinitionForm, families: readonly string[]) {
  const offered = familiesOf(form.entry.categories);
  return offered.every((family) => families.includes(String(family)))
    ? null
    : form.entry.categories.filter((id) => families.includes(String(Math.floor(id / 1000) * 1000)));
}

const sharedOf = (values: AddValues, homeShown: boolean) => ({
  name: values.name.trim(),
  ...(homeShown && { homeNodeId: values.homeNodeId }),
  priority: Number(values.priority),
  searches: limitOf(limitSchema.parse(values.searches)),
  grabs: limitOf(limitSchema.parse(values.grabs)),
  on: values.on,
  automaticSearch: values.automaticSearch,
  rss: values.rss,
});

/** The Indexer the form describes, as POST /indexers and its Test take it. */
export function draftOf(source: Source, values: AddValues, homeShown: boolean): IndexersNewIndexer {
  const shared = sharedOf(values, homeShown);
  const cookie =
    values.cookie.trim() === ""
      ? undefined
      : { cookie: values.cookie.trim(), userAgent: values.userAgent.trim() };
  switch (source.kind) {
    case "catalog":
    case "native": {
      const { form } = source;
      const categories = categoriesOf(form, values.families);
      const address = addressOf(values.baseAddress);
      const settings = {
        values: settingsOf(form.fields, values.values),
        ...(categories !== null && { categories }),
        ...(form.cookie && cookie !== undefined && { cookie }),
      };
      return source.kind === "native"
        ? { ...shared, kind: "native", port: form.entry.id, baseAddress: address, ...settings }
        : {
            ...shared,
            kind: "catalog",
            definition: form.entry.id,
            ...(address !== form.links[0] && { baseAddress: address }),
            ...settings,
          };
    }
    case "torznab":
      return {
        ...shared,
        kind: "torznab",
        address: values.address.trim(),
        apiKey: values.apiKey,
        ...(values.trackerName.trim() !== "" && { trackerName: values.trackerName.trim() }),
      };
    case "rss":
      return {
        ...shared,
        kind: "rss",
        address: values.address.trim(),
        trackerName: values.trackerName.trim(),
        media: values.media,
        allowNoSize: values.allowNoSize,
        ...(cookie !== undefined && { cookie: { cookie: cookie.cookie } }),
      };
    default:
      return source satisfies never;
  }
}

/** Where a refusal's field lands in the form: a setting by its index, an address by its field. */
export function fieldOf(source: Source, location: string): string {
  if (location.startsWith("cookie")) {
    return "cookie";
  }
  const form = formOf(source);
  if (form !== undefined && location.startsWith("values.")) {
    return settingAt(form.fields, location);
  }
  return location === "baseAddress" ? "baseAddress.typed" : location;
}
