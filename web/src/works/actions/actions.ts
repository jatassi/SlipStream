import type {
  SetWorksProfileUnwanted,
  WorksContentType,
  WorksDeleted,
  WorksDeletingFiles,
  WorksOutcomeList,
  WorksProfileChanged,
} from "../../api/gen/model";
import {
  deleteWorks,
  refreshWorks,
  setWorksMonitored,
  setWorksProfile,
  setWorksRoot,
} from "../../api/gen/query/works/works";

type Shapes = {
  readonly monitor: { readonly on: boolean };
  /** `confirm` is the unwanted file count the Admin confirmed from the Impact. */
  readonly profile: {
    readonly profileId: string;
    readonly unwanted: SetWorksProfileUnwanted;
    readonly confirm: number;
  };
  readonly root: { readonly rootId: string };
  readonly refresh: object;
  readonly delete: { readonly files: WorksDeletingFiles };
};

export type ActionKind = keyof Shapes;

/** Something done to one or many Works in one request that answers per Work. */
export type WorkAction<K extends ActionKind = ActionKind> = {
  [P in K]: { readonly kind: P } & Shapes[P];
}[K];

type Answers = {
  readonly monitor: WorksOutcomeList;
  readonly profile: WorksProfileChanged;
  readonly root: WorksOutcomeList;
  readonly refresh: WorksOutcomeList;
  readonly delete: WorksDeleted;
};

export type AnswerOf<K extends ActionKind> = Answers[K];

/** A Work an action names, for its words. */
export type Subject = {
  readonly id: string;
  readonly label: string;
  readonly type: WorksContentType;
};

/** The Works a sentence is about: one Work's label, or a count with its noun. */
export type Named = { readonly words: string; readonly one: boolean };

const be = ({ one }: Named) => (one ? "is" : "are");
const was = ({ one }: Named) => (one ? "was" : "were");
const its = ({ one }: Named) => (one ? "its" : "their");

export type Entry<K extends ActionKind> = {
  readonly post: (workIds: string[], action: WorkAction<K>) => Promise<Answers[K]>;
  /** What the action did to the Works it changed. */
  readonly done: (named: Named, action: WorkAction<K>, answer: Answers[K]) => string;
  /** Why it left Works that already read that way as they were. */
  readonly already: (named: Named, action: WorkAction<K>) => string;
  /** The verb after "Couldn't", with the Works after it. */
  readonly attempt: (action: WorkAction<K>) => string;
};

export const actions: { readonly [K in ActionKind]: Entry<K> } = {
  monitor: {
    post: (workIds, { on }) => setWorksMonitored({ workIds, on }),
    done: (named, { on }) =>
      on ? `Monitored ${named.words}.` : `Stopped monitoring ${named.words}.`,
    already: (named, { on }) =>
      on
        ? `${named.words} ${was(named)} already monitored.`
        : `${named.words} ${was(named)}n't monitored.`,
    attempt: ({ on }) => (on ? "monitor" : "stop monitoring"),
  },
  profile: {
    post: (workIds, { profileId, unwanted, confirm }) =>
      setWorksProfile({ workIds, profileId }, { unwanted, confirm }),
    done: (named) => `Changed the Profile of ${named.words}.`,
    already: (named) => `${named.words} already ${named.one ? "has" : "have"} that Profile.`,
    attempt: () => "change the Profile of",
  },
  root: {
    post: (workIds, { rootId }) => setWorksRoot({ workIds, rootId }),
    done: (named) => `Set the library root of ${named.words}.`,
    already: (named) => `${named.words} ${be(named)} already in that library root.`,
    attempt: () => "set the library root of",
  },
  refresh: {
    post: (workIds) => refreshWorks({ workIds }),
    done: (named) => `Refreshed the metadata of ${named.words}.`,
    already: (named) => `${named.words} ${was(named)} already being refreshed.`,
    attempt: () => "refresh the metadata of",
  },
  delete: {
    post: (workIds, { files }) => deleteWorks({ workIds, files }),
    done: (named, { files }, { plan }) => {
      if (plan.files === 0) {
        return `Deleted ${named.words}.`;
      }
      return files === "delete"
        ? `Deleted ${named.words} and ${its(named)} files.`
        : `Deleted ${named.words}. ${named.one ? "Its" : "Their"} files stay on disk.`;
    },
    already: (named) => `${named.words} ${was(named)} already deleted.`,
    attempt: () => "delete",
  },
};

export function entryOf<K extends ActionKind>(action: WorkAction<K>): Entry<K> {
  return actions[action.kind];
}
