import { fieldErrors, isApiError, sentenceOf } from "../api/errors";
import type {
  MediaserversDraftBody as Draft,
  MediaserversKindView as Kind,
  MediaserversPatchBody as Patch,
  MediaserversServerView as Server,
  NodesNode,
} from "../api/gen/model";
import { type SubmitErrors, submitErrors } from "../ui/form-context";

export const noNode = "none";

/** The server sheet's draft. The API key stays "" on a saved server unless a new one is typed. */
export type ServerValues = {
  kind: string;
  name: string;
  address: string;
  apiKey: string;
  deviceId: string;
  reachingNodeId: string;
  /** A Node id, or noNode when the server runs on no Node. */
  runsOnNodeId: string;
  fallbackAddress: string;
  browserAddress: string;
};

export type NodeChoice = { readonly id: string; readonly name: string };

/** The Nodes that can reach a server: the Controller first, then each Worker holding Media Server link. */
export type Reaching = { readonly controller: NodeChoice; readonly workers: readonly NodeChoice[] };

export function reachingOf(nodes: readonly NodesNode[]): Reaching {
  const controller = nodes.find((node) => node.self);
  if (controller === undefined) {
    throw new Error("the Nodes list has no Controller");
  }
  const workers = nodes.filter(
    (node) =>
      !node.self &&
      node.roles.some((view) => view.role === "media-server-link" && view.state === "holding"),
  );
  return { controller, workers };
}

export const reachingChoices = ({ controller, workers }: Reaching): readonly NodeChoice[] => [
  controller,
  ...workers,
];

/** The runs-on Node when it can reach the server, otherwise the Controller. */
export function defaultReaching(runsOnNodeId: string, reaching: Reaching): string {
  return reaching.workers.some((node) => node.id === runsOnNodeId)
    ? runsOnNodeId
    : reaching.controller.id;
}

export function hostOf(address: string): string | undefined {
  try {
    return new URL(address.trim()).hostname.replaceAll(/^\[|\]$/gu, "").toLowerCase();
  } catch {
    return undefined;
  }
}

/** The Node whose address shares the server address's host. */
export function suggestedRunsOn(address: string, nodes: readonly NodesNode[]): string {
  const host = hostOf(address);
  if (host === undefined) {
    return noNode;
  }
  const found = nodes.find((node) =>
    node.addresses.some((each) => hostOf(`http://${each}`) === host),
  );
  return found?.id ?? noNode;
}

export function isLocalhost(address: string): boolean {
  const host = hostOf(address);
  return host === "localhost" || host === "::1" || (host?.startsWith("127.") ?? false);
}

export const newValues = (kind: Kind, reaching: Reaching): ServerValues => ({
  kind: kind.kind,
  name: kind.label,
  address: "",
  apiKey: "",
  deviceId: "",
  reachingNodeId: reaching.controller.id,
  runsOnNodeId: noNode,
  fallbackAddress: "",
  browserAddress: "",
});

export const valuesOf = (server: Server): ServerValues => ({
  kind: server.kind,
  name: server.name,
  address: server.address,
  apiKey: "",
  deviceId: "",
  reachingNodeId: server.reachingNode.id,
  runsOnNodeId: server.runsOnNode?.id ?? noNode,
  fallbackAddress: server.fallbackAddress ?? "",
  browserAddress: server.browserAddress ?? "",
});

/** The draft as the add call takes it, with only the fields kind takes. */
export function draftOf(values: ServerValues, kind: Kind): Draft {
  const fallback = values.fallbackAddress.trim();
  const browser = values.browserAddress.trim();
  return {
    kind: values.kind,
    name: values.name.trim(),
    address: values.address.trim(),
    reachingNodeId: values.reachingNodeId,
    ...(values.apiKey !== "" && { apiKey: values.apiKey }),
    ...(values.deviceId !== "" && { deviceId: values.deviceId }),
    ...(values.runsOnNodeId !== noNode && { runsOnNodeId: values.runsOnNodeId }),
    ...(fallback !== "" && { fallbackAddress: fallback }),
    ...(kind.browser && browser !== "" && { browserAddress: browser }),
  };
}

const orNull = (value: string): string | null => {
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
};

/** Only what changed from server. */
export function editOf(server: Server, values: ServerValues, kind: Kind): Patch {
  const name = values.name.trim();
  const address = values.address.trim();
  const runsOn = values.runsOnNodeId === noNode ? null : values.runsOnNodeId;
  const fallback = orNull(values.fallbackAddress);
  const browser = orNull(values.browserAddress);
  return {
    ...(name !== server.name && { name }),
    ...(address !== server.address && { address }),
    ...(values.apiKey !== "" && { apiKey: values.apiKey }),
    ...(values.reachingNodeId !== server.reachingNode.id && {
      reachingNodeId: values.reachingNodeId,
    }),
    ...(runsOn !== (server.runsOnNode?.id ?? null) && { runsOnNodeId: runsOn }),
    ...(fallback !== (server.fallbackAddress ?? null) && { fallbackAddress: fallback }),
    ...(kind.browser && browser !== (server.browserAddress ?? null) && { browserAddress: browser }),
  };
}

const fieldOfCode: Readonly<Record<string, keyof ServerValues>> = {
  media_server_name_taken: "name",
  media_server_sign_in_refused: "apiKey",
  media_server_unreached: "address",
  media_server_other_server: "address",
  node_not_media_server_link: "reachingNodeId",
};

/** A refused save as the form shows it: on the field it names, else above the buttons. */
export function refusalOf(error: unknown): SubmitErrors {
  if (!isApiError(error)) {
    return submitErrors(error);
  }
  if (error.errors.length > 0) {
    return { fields: fieldErrors(error) };
  }
  const field = fieldOfCode[error.code];
  return field === undefined
    ? { form: sentenceOf(error), fields: {} }
    : { fields: { [field]: sentenceOf(error) } };
}

export const underDetails = new Set<string>(["fallbackAddress", "browserAddress"]);

export const kindOf = (kinds: readonly Kind[], id: string): Kind => {
  const kind = kinds.find((each) => each.kind === id);
  if (kind === undefined) {
    throw new Error(`the kinds list has no kind ${id}`);
  }
  return kind;
};

/** The kinds this sheet adds: those that take an API key or an administrator's sign-in. */
export const keyKinds = (kinds: readonly Kind[]): readonly Kind[] =>
  kinds.filter((kind) => kind.signIn === "api_key_or_admin");
