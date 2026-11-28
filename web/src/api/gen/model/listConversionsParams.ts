import type { ListConversionsBand } from "./listConversionsBand.ts";

export type ListConversionsParams = {
  /**
   * Only this band; left out, every band
   */
  band?: ListConversionsBand;
  /**
   * Only Conversions placed or forecast on this Node
   */
  node?: string;
  /**
   * Only Works of this Content Type
   */
  type?: string;
  /**
   * Only Works whose title holds this
   * @maxLength 200
   */
  q?: string;
  /**
   * Only this Work's Conversions
   */
  work?: string;
  /**
   * The cursor of the previous page
   */
  cursor?: string;
};
