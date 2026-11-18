import { sentenceOf } from "../api/errors";
import type { SubtitlesSidecar, TargetsWantedLanguage } from "../api/gen/model";
import { markSubtitleBad } from "../api/gen/query/subtitles/subtitles";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import { type WantAt, searchIsOn } from "./work-api";
import { markBadQuestion, markedBadWords } from "./sheet-words";

/**
 * Mark bad asks first, naming the file version. On an uploaded or found
 * subtitle it says the Admin's own file is deleted for good.
 */
export function MarkBadDialog({
  at,
  wanted,
  sidecar,
  version,
  stem,
  open,
  onOpenChange,
}: {
  at: WantAt;
  wanted: TargetsWantedLanguage;
  sidecar: SubtitlesSidecar;
  /** The file version in words, such as "the 2160p file". */
  version: string;
  /** The video's name without its extension, which a subtitle file's name follows. */
  stem: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { title, body } = markBadQuestion(wanted, sidecar, { version, stem });
  return (
    <ConfirmDialog
      title={title}
      confirmLabel="Mark bad"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        let searchOn: boolean;
        try {
          await markSubtitleBad(at.fileId, at.want);
          searchOn = await searchIsOn();
        } catch (error) {
          throw new Error(sentenceOf(error), { cause: error });
        }
        toast(markedBadWords(wanted, searchOn));
      }}
    >
      <p>{body}</p>
    </ConfirmDialog>
  );
}
