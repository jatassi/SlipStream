import { useEffect, useSyncExternalStore } from "react";

import { DialogActions } from "../account/dialog-actions";
import { sentenceOf } from "../api/errors";
import type { SubtitlesProvider, TargetsWantedLanguage } from "../api/gen/model";
import { startSubtitleCandidates } from "../api/gen/query/subtitles/subtitles";
import { createStore } from "../shell/store";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Sheet } from "../ui/sheet";
// oxlint-disable-next-line import/max-dependencies -- the host starts, frames and fills both subtitle sheets on either shell
import { useShell } from "../ui/shell-context";
import { Skeleton } from "../ui/skeleton";
import { CandidatesBody } from "./candidates";
import { UploadBody } from "./upload";
import type { WantAt } from "./work-api";
import { sheetTitle } from "./sheet-words";

type Start =
  | { readonly state: "starting" }
  | { readonly state: "started"; readonly searchId: string }
  | { readonly state: "refused"; readonly sentence: string };

/** What a subtitle sheet is about: one want of one file, by the file's name. */
export type SubtitleSubject = WantAt & {
  readonly wanted: TargetsWantedLanguage;
  readonly fileName: string;
  /** The Subtitle Providers, which name each candidate's source. */
  readonly providers: readonly SubtitlesProvider[];
};

export type SubtitleOpening = {
  /** Tells one opening from the next, so a late answer to an earlier start is dropped. */
  readonly serial: number;
  readonly subject: SubtitleSubject;
  readonly mode: "search" | "upload";
  readonly start: Start;
};

type HostState =
  | { readonly shown: "none" }
  | { readonly shown: "open" | "closing"; readonly opening: SubtitleOpening };

type HostEvent =
  | { readonly kind: "open"; readonly opening: SubtitleOpening }
  | { readonly kind: "settle"; readonly serial: number; readonly start: Start }
  | { readonly kind: "close" };

const host = createStore<HostState, HostEvent>(
  (state, event) => {
    if (event.kind === "open") {
      return { shown: "open", opening: event.opening };
    }
    if (state.shown === "none") {
      return state;
    }
    if (event.kind === "close") {
      return state.shown === "open" ? { shown: "closing", opening: state.opening } : state;
    }
    return state.opening.serial === event.serial
      ? { ...state, opening: { ...state.opening, start: event.start } }
      : state;
  },
  { shown: "none" },
);

let serials = 0;

async function start(serial: number, { fileId, want }: SubtitleSubject) {
  let started: Start;
  try {
    const { searchId } = await startSubtitleCandidates(fileId, want);
    started = { state: "started", searchId };
  } catch (error) {
    started = { state: "refused", sentence: sentenceOf(error) };
  }
  host.dispatch({ kind: "settle", serial, start: started });
}

/** Opens Manual subtitle search on one want of one file and starts its search. */
export function openSubtitleSearch(subject: SubtitleSubject) {
  serials += 1;
  const serial = serials;
  host.dispatch({
    kind: "open",
    opening: { serial, subject, mode: "search", start: { state: "starting" } },
  });
  void start(serial, subject);
}

/** Opens the upload dialog for one want of one file. */
export function openSubtitleUpload(subject: SubtitleSubject) {
  serials += 1;
  host.dispatch({
    kind: "open",
    opening: { serial: serials, subject, mode: "upload", start: { state: "starting" } },
  });
}

export const closeSubtitleSheet = () => {
  host.dispatch({ kind: "close" });
};

const useHost = (): HostState => useSyncExternalStore(host.subscribe, host.getSnapshot);

function Refused({ sentence }: { sentence: string }) {
  return (
    <div className="flex flex-col gap-4">
      <AlertLine text={sentence} />
      <DialogActions>
        <Button onClick={closeSubtitleSheet}>Close</Button>
      </DialogActions>
    </div>
  );
}

function Opened({ opening }: { opening: SubtitleOpening }) {
  if (opening.mode === "upload") {
    return <UploadBody subject={opening.subject} onClose={closeSubtitleSheet} />;
  }
  const { start: started } = opening;
  if (started.state === "refused") {
    return <Refused sentence={started.sentence} />;
  }
  if (started.state === "starting") {
    return <Skeleton label="Starting the search" />;
  }
  return (
    <CandidatesBody
      subject={opening.subject}
      searchId={started.searchId}
      onClose={closeSubtitleSheet}
    />
  );
}

const onOpenChange = (next: boolean) => {
  if (!next) {
    closeSubtitleSheet();
  }
};

function Framed({ opening, open }: { opening: SubtitleOpening; open: boolean }) {
  const shell = useShell();
  const { wanted, fileName } = opening.subject;
  const title = sheetTitle(wanted, fileName);
  const body = <Opened opening={opening} />;
  if (opening.mode === "upload") {
    return (
      <Dialog title={title} open={open} onOpenChange={onOpenChange}>
        {body}
      </Dialog>
    );
  }
  return shell === "phone" ? (
    <Sheet title={title} fill open={open} onOpenChange={onOpenChange}>
      {body}
    </Sheet>
  ) : (
    <Dialog title={title} size={940} open={open} onOpenChange={onOpenChange}>
      {body}
    </Dialog>
  );
}

/**
 * Manual subtitle search and Upload for one want of one file: a 940 dialog
 * on desktop and a full sheet on phone for the search, and a dialog, a
 * bottom sheet on phone, for an upload. Mounted beside Manual search.
 */
export function SubtitleSearchHost() {
  const state = useHost();
  // A sheet left open when the session ends must not greet the next one.
  useEffect(() => closeSubtitleSheet, []);
  return state.shown === "none" ? null : (
    <Framed key={state.opening.serial} opening={state.opening} open={state.shown === "open"} />
  );
}
