import type { TargetsCut, WorksContentType } from "../api/gen/model";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Details } from "../ui/details";
import { type Column, Table } from "../ui/table";
import { CutMenu } from "./cut-actions";
import { AddCut } from "./cut-dialogs";
import { isCustom, originOf, readAsOf } from "./cuts";

function columnsOf(
  types: readonly WorksContentType[],
): readonly [Column<TargetsCut>, ...Column<TargetsCut>[]] {
  return [
    {
      id: "name",
      header: "Name",
      cell: (cut) => <span className="font-medium">{cut.name}</span>,
    },
    { id: "aliases", header: "Read in titles as", cell: readAsOf },
    { id: "origin", header: "Comes from", cell: (cut) => originOf(cut, types) },
    {
      id: "actions",
      header: "Actions",
      align: "end",
      cell: (cut) => (isCustom(cut) ? <CutMenu cut={cut} /> : null),
    },
  ];
}

export function EditionsSection() {
  const { cuts } = useListCutsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="max-w-[64ch]">
          <Details summary="A cut is a version of a Work, such as Extended or the Director's Cut. SlipStream reads it from release titles and file names. Add a cut when your releases name one it doesn't know.">
            <p className="max-w-[64ch]">
              Each phrase is matched as whole words, the longest first. A title that names no cut is
              Standard. Built-in cuts and the ones a Content Type adds can&apos;t be changed.
              Renaming a cut renames its files.
            </p>
          </Details>
        </div>
        <AddCut />
      </div>
      <Table
        label="Cuts"
        columns={columnsOf(contentTypes)}
        rows={cuts}
        rowKey={(cut) => cut.key}
        stackBelow={560}
      />
    </div>
  );
}
