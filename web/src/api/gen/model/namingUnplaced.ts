import type { NamingUnplacedCode } from "./namingUnplacedCode.ts";

export interface NamingUnplaced {
  /** Why the Content Type can't tell which Item the name holds */
  code: NamingUnplacedCode;
  /** For no_such_item: the Item the name names, such as S03E14 */
  names?: string;
}
