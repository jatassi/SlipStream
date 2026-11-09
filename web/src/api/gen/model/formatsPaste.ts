import type { FormatsPasteApp } from "./formatsPasteApp.ts";

export interface FormatsPaste {
  /** The app the text was written for; left out, the app its kinds settle: Edition, Quality modifier or Year mean Radarr and Release type means Sonarr */
  app?: FormatsPasteApp;
  /**
   * The Custom Formats of a pasted one's name that it replaces; a pasted one whose name is taken and not listed here is added beside the other, with a number after its name
   * @nullable
   */
  replace?: string[] | null;
  /**
   * One Custom Format's JSON or a list of them, as TRaSH Guides, Radarr and Sonarr export them or as Sonarr's API returns them
   * @minLength 1
   */
  text: string;
}
