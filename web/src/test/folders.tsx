import { QueryClientProvider } from "@tanstack/react-query";
import { HttpResponse, http } from "msw";
import type { ReactNode } from "react";

import type {
  RootsFolderProblem,
  RootsFolderProblemKind,
  RootsListing,
  RootsPlace,
  RootsSegment,
} from "../api/gen/model";
import { MakeFolderBody } from "../api/gen/zod/roots/roots.zod";
import { createQueryClient } from "../api/query-client";
import { renderKit } from "./render-kit";
import { worker } from "./worker";

export type FakeFolder = {
  readonly name: string;
  readonly readable?: boolean;
  readonly hidden?: boolean;
};

/** A folder on a fake Node: its subfolders, or the error reading it answers. */
export type FakeEntry =
  | {
      readonly path: string;
      readonly parent: string;
      readonly segments: readonly RootsSegment[];
      readonly folders: FakeFolder[];
      readonly join: (name: string) => string;
    }
  | { readonly problem: RootsFolderProblem };

const named = (names: readonly (string | FakeFolder)[]) =>
  names.map((name) => (typeof name === "string" ? { name } : name));

/** A folder on a Unix Node, split as the Node would split it. */
export function unixFolder(path: string, names: readonly (string | FakeFolder)[] = []): FakeEntry {
  const parts = path.split("/").filter(Boolean);
  const segments = [
    { name: "/", path: "/" },
    ...parts.map((name, index) => ({ name, path: `/${parts.slice(0, index + 1).join("/")}` })),
  ];
  const join = (name: string) => (path === "/" ? `/${name}` : `${path}/${name}`);
  return { path, parent: segments.at(-2)?.path ?? "", segments, folders: named(names), join };
}

/** A folder on a Windows Node under a drive such as D:\. */
export function windowsFolder(
  path: string,
  names: readonly (string | FakeFolder)[] = [],
): FakeEntry {
  const [drive = "", ...parts] = path.split("\\").filter(Boolean);
  const root = `${drive}\\`;
  const segments = [
    { name: root, path: root },
    ...parts.map((name, index) => ({
      name,
      path: `${root}${parts.slice(0, index + 1).join("\\")}`,
    })),
  ];
  const join = (name: string) => (path.endsWith("\\") ? `${path}${name}` : `${path}\\${name}`);
  return { path, parent: segments.at(-2)?.path ?? "", segments, folders: named(names), join };
}

export const unreadable = (
  kind: RootsFolderProblemKind,
  osText: string,
  account?: string,
): FakeEntry => ({
  problem: { kind, osText, ...(account !== undefined && { account }) },
});

export type FakeNode = {
  readonly places: readonly RootsPlace[];
  readonly service?: boolean;
  /** Each folder by the path the Node cleans it to. */
  readonly folders: Record<string, FakeEntry>;
  /** Typed paths and the folder the Node cleans each to, such as d:/media to D:\Media. */
  readonly cleans?: Readonly<Record<string, string>>;
  readonly suggests?: Readonly<Record<string, readonly string[]>>;
};

const problemType = { "Content-Type": "application/problem+json" };

const refusal = (status: number, code: string, title: string, members: object = {}) =>
  HttpResponse.json({ status, code, title, ...members }, { status, headers: problemType });

function listingOf(
  node: FakeNode,
  path: string,
  hidden: boolean,
): RootsListing | RootsFolderProblem {
  if (path === "") {
    return {
      path: "",
      parent: "",
      segments: [],
      folders: [],
      hidden: 0,
      places: [...node.places],
      service: node.service ?? false,
    };
  }
  const entry =
    node.folders[node.cleans?.[path] ?? path] ??
    unreadable("not_found", `stat ${path}: no such file or directory`);
  if ("problem" in entry) {
    return entry.problem;
  }
  const shown = entry.folders.filter((folder) => hidden || folder.hidden !== true);
  return {
    path: entry.path,
    parent: entry.parent,
    segments: [...entry.segments],
    folders: shown.map((folder) => ({
      name: folder.name,
      path: entry.join(folder.name),
      readable: folder.readable ?? true,
    })),
    hidden: entry.folders.length - shown.length,
    places: [],
    service: node.service ?? false,
  };
}

/** Serves a fake Node's folders, and answers node_offline for `offline`; the Node's folders change as New folder makes them. */
export function serveFolders(nodes: Readonly<Record<string, FakeNode>>, offline?: string) {
  const asked: string[] = [];
  worker.use(
    http.get("*/api/v1/nodes/:id/folders", ({ params, request }) => {
      const url = new URL(request.url);
      const path = url.searchParams.get("path") ?? "";
      asked.push(path);
      const node = nodes[String(params["id"])];
      if (node === undefined || params["id"] === offline) {
        return refusal(503, "node_offline", "The Node is offline");
      }
      const listing = listingOf(node, path, url.searchParams.get("hidden") === "true");
      return "kind" in listing
        ? refusal(422, "folder_unreadable", "The Node can't read the folder", { problem: listing })
        : HttpResponse.json(listing);
    }),
    http.get("*/api/v1/nodes/:id/folders/suggest", ({ params, request }) => {
      const text = new URL(request.url).searchParams.get("text") ?? "";
      return HttpResponse.json({
        suggestions: [...(nodes[String(params["id"])]?.suggests?.[text] ?? [])],
      });
    }),
    http.post("*/api/v1/nodes/:id/folders", async ({ params, request }) => {
      const { parent, name } = MakeFolderBody.parse(await request.json());
      const node = nodes[String(params["id"])];
      const entry = node?.folders[parent];
      if (node === undefined || entry === undefined || "problem" in entry) {
        return refusal(422, "folder_invalid", "The Node can't use that path or folder name");
      }
      if (entry.folders.some((folder) => folder.name === name)) {
        return refusal(409, "folder_exists", "A folder of that name is already here");
      }
      const path = entry.join(name);
      entry.folders.push({ name });
      node.folders[path] = {
        ...entry,
        path,
        parent: entry.path,
        segments: [...entry.segments, { name, path }],
        folders: [],
        join: (child: string) => `${path}${path.includes("\\") ? "\\" : "/"}${child}`,
      };
      return HttpResponse.json({ path }, { status: 201 });
    }),
  );
  return asked;
}

/** Renders kit UI that reads the server through TanStack Query. */
export function renderQueried(ui: ReactNode, options?: Parameters<typeof renderKit>[1]) {
  const client = createQueryClient(() => {});
  return renderKit(<QueryClientProvider client={client}>{ui}</QueryClientProvider>, options);
}
