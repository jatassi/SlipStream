import { Facts } from "../formats/tested-name";
import { AlertLine } from "../ui/alert-line";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { type TestedFile, leadOf, pathOf } from "./tester-words";

export type Shown =
  | { readonly state: "idle" }
  | { readonly state: "results"; readonly files: readonly TestedFile[] }
  | { readonly state: "refused"; readonly sentence: string };

function Tested({ file }: { file: TestedFile }) {
  const path = pathOf(file.outcome);
  return (
    <li className="flex flex-col gap-1 border-t border-hairline py-3 text-body first:border-t-0 first:pt-0 in-data-[density=touch]:text-body-touch">
      <Data className="wrap-anywhere text-ink-3">{file.name}</Data>
      <Details
        summary={leadOf(file.outcome)}
        action={path === undefined ? undefined : <Data className="wrap-anywhere">{path}</Data>}
      >
        <div className="mt-1">
          <Facts facts={file.facts} />
        </div>
      </Details>
    </li>
  );
}

export function Results({ shown }: { shown: Shown }) {
  switch (shown.state) {
    case "idle":
      return null;
    case "refused":
      return <AlertLine text={shown.sentence} />;
    case "results":
      return (
        <ol aria-label="Results" className="flex flex-col">
          {shown.files.map((file, index) => (
            // Two pasted lines can be the same name.
            <Tested key={`${String(index)} ${file.name}`} file={file} />
          ))}
        </ol>
      );
    default:
      return shown satisfies never;
  }
}
