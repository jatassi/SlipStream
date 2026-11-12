import { useGetImportSuspense } from "../api/gen/query/imports/imports";
import { FileLine } from "../imports/file-line";
import { ImportFilesSheet } from "../imports/import-files-sheet";

/** Each file of a Grab's download: where it got to and why. */
export function ImportLines({ importId }: { importId: string }) {
  const { files } = useGetImportSuspense(importId).data;
  return (
    <section aria-label="Files">
      <ul className="flex flex-col gap-3">
        {files.map((file) => (
          <FileLine key={file.position} file={file} />
        ))}
      </ul>
    </section>
  );
}

/** The Import files sheet on a Grab's own Import. */
export function ImportByHand({ importId, onClose }: { importId: string; onClose: () => void }) {
  return <ImportFilesSheet start={{ importIds: [importId], source: "grab" }} onClose={onClose} />;
}
