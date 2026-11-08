import type { FormatsDraft } from "./formatsDraft.ts";
import type { FormatsParseKind } from "./formatsParseKind.ts";

export interface FormatsParse {
  /** A Custom Format matched as if saved, in place of formatId's */
  draft?: FormatsDraft;
  /** release reads each name as a release title; file reads it as a file path, with its folder's name when the file's gives no title */
  kind?: FormatsParseKind;
  /**
   * Release names, or file paths when kind is file
   * @minItems 1
   * @maxItems 100
   * @nullable
   */
  names: string[] | null;
  /** The Work to judge every name against; left out, each name is matched to the Works whose titles it gives, read as each Content Type in turn */
  workId?: string;
}
