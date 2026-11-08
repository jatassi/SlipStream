export interface DownloadclientsPollProblem {
  /** Why the client isn't responding, refused the sign-in or needs attention */
  cause: string;
  /** The form fields an Admin changes to fix it */
  fields: string[];
  /** What happened and what to do, as a sentence */
  sentence: string;
}
