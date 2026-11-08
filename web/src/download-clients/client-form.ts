import { z } from "zod";

import { type ApiError, fieldErrors, isApiError, isProblem, sentenceOf } from "../api/errors";
import type {
  DownloadclientsClientEdit as ClientEdit,
  DownloadclientsDaemon as Daemon,
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
  DownloadclientsNewClient as NewClient,
  DownloadclientsTestReport as TestReport,
} from "../api/gen/model";
import { type SubmitErrors, submitErrors } from "../ui/form-context";

export type KindId = Kind["kind"];

export type FolderPair = { client: string; node: string };

/** The client sheet's draft. A secret stays "" unless a new one is typed, so an empty one keeps the saved value. */
export type ClientValues = {
  on: boolean;
  nodeId: string;
  kind: KindId;
  name: string;
  address: string;
  username: string;
  password: string;
  apiKey: string;
  label: string;
  folder: string;
  clientFolders: FolderPair[];
  /** host:port, or "" for the only daemon. */
  daemon: string;
};

export const examples: { readonly [K in KindId]: string } = {
  qbittorrent: "http://192.168.1.20:8080",
  transmission: "http://192.168.1.20:9091/transmission/rpc",
  deluge: "http://192.168.1.20:8112/json",
  rtorrent: "http://192.168.1.20/RPC2",
};

export const signInOf = (kind: Kind) => ({
  username: kind.username !== "none",
  password: kind.password !== "none",
  apiKey: kind.apiKey,
});

export const kindOf = (kinds: readonly Kind[], id: KindId): Kind => {
  const kind = kinds.find((each) => each.kind === id);
  if (kind === undefined) {
    throw new Error(`the Listing has no kind ${id}`);
  }
  return kind;
};

export const daemonKey = ({ host, port }: Daemon): string => `${host}:${port}`;

const daemonOf = (key: string): Daemon | undefined => {
  const at = key.lastIndexOf(":");
  return at === -1 ? undefined : { host: key.slice(0, at), port: Number(key.slice(at + 1)) };
};

export const newValues = (nodeId: string, kind: Kind): ClientValues => ({
  on: true,
  nodeId,
  kind: kind.kind,
  name: kind.name,
  address: "",
  username: "",
  password: "",
  apiKey: "",
  label: "slipstream",
  folder: "",
  clientFolders: [],
  daemon: "",
});

export const valuesOf = (client: External): ClientValues => ({
  on: client.on,
  nodeId: client.nodeId,
  kind: client.kind,
  name: client.name,
  address: client.address,
  username: client.username,
  password: "",
  apiKey: "",
  label: client.label,
  folder: client.folder,
  clientFolders: client.clientFolders.map((pair) => ({ ...pair })),
  daemon: client.daemon === null ? "" : daemonKey(client.daemon),
});

const pairsOf = (pairs: readonly FolderPair[]): FolderPair[] =>
  pairs
    .map(({ client, node }) => ({ client: client.trim(), node: node.trim() }))
    .filter(({ client, node }) => client !== "" || node !== "");

/** The draft as the create and Test calls take it, with only the fields kind takes. */
export function draftOf(values: ClientValues, kind: Kind): NewClient {
  const signIn = signInOf(kind);
  const daemon = kind.daemon ? daemonOf(values.daemon) : undefined;
  const name = values.name.trim();
  const username = values.username.trim();
  const folder = values.folder.trim();
  return {
    nodeId: values.nodeId,
    kind: values.kind,
    address: values.address.trim(),
    label: values.label.trim(),
    on: values.on,
    clientFolders: pairsOf(values.clientFolders),
    ...(name !== "" && { name }),
    ...(signIn.username && username !== "" && { username }),
    ...(signIn.password && values.password !== "" && { password: values.password }),
    ...(signIn.apiKey && values.apiKey !== "" && { apiKey: values.apiKey }),
    ...(folder !== "" && { folder }),
    ...(daemon !== undefined && { daemon }),
  };
}

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** Only what changed from client, so a PATCH tests the client only when a setting that reaches it changed. */
export function editOf(client: External, values: ClientValues, kind: Kind): ClientEdit {
  const draft = draftOf(values, kind);
  const name = draft.name ?? "";
  const username = draft.username ?? "";
  const label = draft.label ?? "";
  const folder = draft.folder ?? "";
  const pairs = draft.clientFolders ?? [];
  const daemon = draft.daemon ?? null;
  return {
    ...(draft.nodeId !== client.nodeId && { nodeId: draft.nodeId }),
    ...(name !== client.name && { name }),
    ...(draft.address !== client.address && { address: draft.address }),
    ...(username !== client.username && { username }),
    ...(draft.password !== undefined && { password: draft.password }),
    ...(draft.apiKey !== undefined && { apiKey: draft.apiKey }),
    ...(label !== client.label && { label }),
    ...(folder !== client.folder && { folder }),
    ...(!same(pairs, client.clientFolders) && { clientFolders: pairs }),
    ...(!same(daemon, client.daemon) && { daemon }),
    ...(draft.on !== client.on && { on: draft.on }),
  };
}

const Marked = z.object({ fields: z.array(z.string()).nullish() });

export const marking = "Change this, then test again.";

/** The fields a failed Test names, marked with what to do. */
export const markedOf = (failed: ApiError): SubmitErrors => ({
  fields: Object.fromEntries(
    (Marked.parse(failed.body).fields ?? []).map((field) => [field, marking]),
  ),
});

const whole = new Set([
  "download_client_unavailable",
  "download_client_in_use",
  "download_client_has_torrents",
]);

/** A refused save or Test, other than a failed Test, as the form shows it. */
export function refusalOf(error: unknown): SubmitErrors {
  if (isProblem(error, "download_client_field_refused")) {
    return { fields: fieldErrors(error) };
  }
  if (isProblem(error, "download_client_address_taken")) {
    return { fields: { address: sentenceOf(error) } };
  }
  if (isProblem(error, "download_client_name_taken")) {
    return { fields: { name: error.title } };
  }
  if (isProblem(error, "node_not_downloading")) {
    return { fields: { nodeId: error.title } };
  }
  if (isApiError(error) && whole.has(error.code)) {
    return { form: sentenceOf(error), fields: {} };
  }
  return submitErrors(error);
}

export type TestResult =
  | { readonly passed: true; readonly report: TestReport }
  | { readonly passed: false; readonly sentence: string; readonly daemons: readonly Daemon[] };

const Listed = z.object({
  daemons: z.array(z.object({ host: z.string(), port: z.number() })).nullish(),
});

/**
 * What a refused save or Test shows: a failed Test's sentence on the result
 * line with the fields it names marked, Deluge's daemons to choose from when
 * its web UI lists more than one, or else the refusal.
 */
export function answerOf(error: unknown): { result?: TestResult; errors: SubmitErrors } {
  if (isProblem(error, "download_client_choose_daemon")) {
    return {
      result: {
        passed: false,
        sentence: sentenceOf(error),
        daemons: Listed.parse(error.body).daemons ?? [],
      },
      errors: { fields: {} },
    };
  }
  return isProblem(error, "download_client_test_failed")
    ? {
        result: { passed: false, sentence: sentenceOf(error), daemons: [] },
        errors: markedOf(error),
      }
    : { errors: refusalOf(error) };
}

/** The daemons the last Test listed, passed or not. */
export const daemonsOf = (result: TestResult | undefined): readonly Daemon[] => {
  if (result === undefined) {
    return [];
  }
  return result.passed ? result.report.daemons : result.daemons;
};

const underDetails = new Set(["label", "folder", "clientFolders", "daemon"]);

/** Whether errors name a field the Details disclosure holds. */
export const namesDetails = (errors: SubmitErrors): boolean =>
  Object.keys(errors.fields).some((field) => underDetails.has(field.replace(/\[.*$/u, "")));
