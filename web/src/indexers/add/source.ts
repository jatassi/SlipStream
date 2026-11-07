import type {
  IndexersDefinitionField,
  IndexersDefinitionForm,
  IndexersNewIndexer,
  NodesNode,
  WorksContentType,
} from "../../api/gen/model";
import { type LimitValues, noLimit } from "../fields/limit-field";
import { limitOf, limitSchema } from "../edit-values";
import { familiesOf } from "./families";

/** What a new Indexer is added from: a definition, or a feed of a kind. */
export type Source =
  | { readonly kind: "catalog"; readonly form: IndexersDefinitionForm }
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
  baseAddress: string;
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
};

/** A definition field that holds a value, which an info field and the captcha don't. */
export const holdsValue = ({ type }: IndexersDefinitionField) =>
  type === "text" || type === "password" || type === "checkbox" || type === "select";

function startValue(field: IndexersDefinitionField): string {
  if (field.type === "checkbox") {
    return field.default === "true" ? "true" : "false";
  }
  if (field.type === "select") {
    return field.default ?? field.options[0]?.value ?? "";
  }
  return holdsValue(field) ? (field.default ?? "") : "";
}

export function startValues(source: Source, home: NodesNode | undefined, name = ""): AddValues {
  const form = source.kind === "catalog" ? source.form : undefined;
  const enabled = source.kind === "rss" ? source.contentTypes.filter((type) => type.enabled) : [];
  return {
    name: form?.entry.name ?? name,
    homeNodeId: home?.id ?? "",
    priority: "25",
    searches: noLimit,
    grabs: noLimit,
    on: true,
    automaticSearch: true,
    rss: true,
    baseAddress: form?.links[0] ?? "",
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
  };
}

/** The username a captcha is shown for, which a new one needs once it changes. */
export function signInOf(source: Source, values: AddValues): string {
  if (source.kind !== "catalog") {
    return "";
  }
  const username = source.form.fields.findIndex((field) => field.name === "username");
  return `${values.homeNodeId}\n${username < 0 ? "" : values.values[username]}`;
}

/** The feed a Torznab Test read, which a new Test needs once it changes. */
export const feedOf = (values: AddValues) => `${values.address.trim()}\n${values.apiKey}`;

function settingsOf(form: IndexersDefinitionForm, values: AddValues) {
  return Object.fromEntries(
    form.fields.flatMap((field, index) => {
      const value = values.values[index] ?? "";
      return holdsValue(field) && value !== "" ? [[field.name, value]] : [];
    }),
  );
}

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
    case "catalog": {
      const { form } = source;
      const categories = categoriesOf(form, values.families);
      return {
        ...shared,
        kind: "catalog",
        definition: form.entry.id,
        ...(values.baseAddress !== form.links[0] && { baseAddress: values.baseAddress }),
        values: settingsOf(form, values),
        ...(categories !== null && { categories }),
        ...(form.cookie && cookie !== undefined && { cookie }),
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

/** Where a refusal's field lands in the form: a setting by its index. */
export function fieldOf(source: Source, location: string): string {
  if (location.startsWith("cookie")) {
    return "cookie";
  }
  if (source.kind === "catalog" && location.startsWith("values.")) {
    const index = source.form.fields.findIndex((field) => `values.${field.name}` === location);
    return index < 0 ? location : `values[${index}]`;
  }
  return location;
}
