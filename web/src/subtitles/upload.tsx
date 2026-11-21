import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { sentenceOf } from "../api/errors";
import { keepSubtitleUpload, uploadSubtitle } from "../api/gen/query/subtitles/subtitles";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { FileField } from "../ui/file-field";
import { SignalLine } from "../ui/signal-line";
import { toast } from "../ui/toast";
import { AttemptLine, useAttemptLine } from "./attempt";
// oxlint-disable-next-line import/max-dependencies -- the upload checks, sends and follows one file, and keeps it on request
import type { SubtitleSubject } from "./search-host";
import { addedWords, keepAnywayWords, tooLargeWords, uploadFailedWords } from "./sheet-words";

// The server's UploadCap, checked here first so a large file is never sent.
const uploadCap = 5 << 20;

const formats = ".srt,.ass,.ssa,.vtt,.sub,.smi";

// btoa takes a binary string; a chunk keeps the spread under the argument limit.
const chunk = 0x8000;

async function base64Of(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  let binary = "";
  for (let at = 0; at < bytes.length; at += chunk) {
    binary += String.fromCodePoint(...bytes.subarray(at, at + chunk));
  }
  return btoa(binary);
}

function KeepAnyway({
  subject,
  uploadId,
  match,
  threshold,
  onKept,
}: {
  subject: SubtitleSubject;
  uploadId: string;
  match: number | null;
  threshold: number | null;
  onKept: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const keep = async () => {
    setBusy(true);
    try {
      await keepSubtitleUpload(subject.fileId, uploadId);
      toast(addedWords(subject.wanted));
      onKept();
    } catch (error) {
      setFailure(sentenceOf(error));
      setBusy(false);
    }
  };
  return (
    <Details summary={uploadFailedWords}>
      <div className="flex flex-col items-start gap-3 pb-1">
        <p>{keepAnywayWords(match, threshold)}</p>
        {failure !== undefined && (
          <SignalLine tone="fail" announce>
            {failure}
          </SignalLine>
        )}
        <Button busy={busy} onClick={() => void keep()}>
          Keep anyway
        </Button>
      </div>
    </Details>
  );
}

function useUpload(subject: SubtitleSubject, onClose: () => void) {
  const [refusal, setRefusal] = useState<string>();
  const [attemptId, setAttemptId] = useState<string>();
  const line = useAttemptLine(subject, attemptId, onClose);
  const send = async (file: File) => {
    setAttemptId(undefined);
    if (file.size > uploadCap) {
      setRefusal(tooLargeWords);
      return;
    }
    setRefusal(undefined);
    try {
      const sent = await uploadSubtitle(subject.fileId, subject.want, {
        name: file.name,
        bytes: await base64Of(file),
      });
      setAttemptId(sent.attemptId);
    } catch (error) {
      setRefusal(sentenceOf(error));
    }
  };
  return { refusal, line, send };
}

/**
 * Upload a subtitle file for one want: a file of 5 MB or smaller, picked or
 * dropped, is sent and checked against the file's timing. A failing upload
 * offers Keep anyway under Details.
 */
export function UploadBody({
  subject,
  onClose,
}: {
  subject: SubtitleSubject;
  onClose: () => void;
}) {
  const { refusal, line, send } = useUpload(subject, onClose);
  const ended = line.attempt?.outcome ?? null;
  const uploadId = ended?.kind === "failed_timing" ? (line.attempt?.uploadId ?? null) : null;
  return (
    <div className="flex flex-col gap-4">
      <FileField
        label="Subtitle file"
        accept={formats}
        description="A subtitle file of 5 MB or smaller, such as SubRip or ASS. Drop it here or choose it."
        onFile={send}
      />
      {refusal !== undefined && (
        <SignalLine tone="fail" announce>
          {refusal}
        </SignalLine>
      )}
      {ended !== null && uploadId !== null ? (
        <KeepAnyway
          subject={subject}
          uploadId={uploadId}
          match={ended.match}
          threshold={ended.threshold}
          onKept={onClose}
        />
      ) : (
        <AttemptLine line={line} providers={subject.providers} />
      )}
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </div>
  );
}
