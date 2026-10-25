import type {
  RootsMissingFileReason,
  RootsRefusal,
  RootsRefusalReason,
  RootsRoot,
  RootsRootState,
  RootsVerdict,
  WorksContentType,
} from "../api/gen/model";
import { bytes } from "../format";
import type { FolderVerdict } from "../folders/verdict";

export type StateTone = "ready" | "quiet" | "warn";

/** A root's state as its row says it, given its Node's name. */
export const stateWords: {
  readonly [S in RootsRootState]: {
    readonly tone: StateTone;
    readonly words: (node: string) => string;
  };
} = {
  ready: { tone: "ready", words: () => "Ready" },
  checking: { tone: "quiet", words: (node) => `Waiting for ${node} to check the folder` },
  missing: { tone: "warn", words: () => "The folder isn't there" },
  unmounted: { tone: "warn", words: () => "The folder is there but looks unmounted" },
  foreign: { tone: "warn", words: () => "The folder holds another library root's marker" },
  unwritable: { tone: "warn", words: () => "SlipStream can't write to the folder" },
  unresponsive: { tone: "warn", words: () => "The folder isn't responding" },
  offline: { tone: "warn", words: (node) => `${node} is offline` },
  written_off: { tone: "warn", words: () => "Node written off" },
};

export const missingWords: { readonly [R in RootsMissingFileReason]: string } = {
  missing: "not there",
  size_differs: "a different size",
};

export type Namer = {
  readonly node: string;
  readonly roots: readonly RootsRoot[];
  readonly types: readonly WorksContentType[];
};

export const typeLabel = (types: readonly WorksContentType[], id: string): string =>
  types.find((type) => type.id === id)?.work.many ?? id;

function rootNamed(rootId: string | undefined, { roots, types }: Namer): string {
  const root = roots.find((candidate) => candidate.id === rootId);
  return root === undefined
    ? "another library root"
    : `the ${typeLabel(types, root.contentType)} library root`;
}

const because: {
  readonly [R in RootsRefusalReason]: (refusal: RootsRefusal, namer: Namer) => string;
} = {
  not_absolute: () => "it isn't a full path",
  not_found: () => "it doesn't exist",
  not_a_folder: () => "it isn't a folder",
  unwritable: () => "SlipStream can't write there",
  inside_root: (refusal, namer) => `it's inside ${rootNamed(refusal.rootId, namer)}`,
  contains_root: (refusal, namer) => `it contains ${rootNamed(refusal.rootId, namer)}`,
  inside_data: () => "it's inside SlipStream's data folder",
  contains_data: () => "it contains SlipStream's data folder",
  inside_downloads: () => "it's inside the downloads folder",
  contains_downloads: () => "it contains the downloads folder",
  unresponsive: (_refusal, { node }) => `${node} didn't respond in time`,
  node_offline: (_refusal, { node }) => `${node} is offline`,
};

export const refusalSentence = (refusal: RootsRefusal, namer: Namer): string =>
  `Can't use this folder because ${because[refusal.reason](refusal, namer)}.`;

export function verdictOf(verdict: RootsVerdict, namer: Namer): FolderVerdict {
  if (verdict.refusal !== undefined) {
    return { usable: false, sentence: refusalSentence(verdict.refusal, namer) };
  }
  const { volume } = verdict;
  return {
    usable: true,
    sentence:
      volume === undefined
        ? "SlipStream can use this folder."
        : `This folder is on ${volume.name}, ${bytes(volume.free)} free.`,
  };
}
