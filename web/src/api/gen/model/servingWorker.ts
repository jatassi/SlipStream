export interface ServingWorker {
  /** Its Controller's name */
  controller: string;
  /** This Worker's key fingerprint, as its Controller lists it */
  fingerprint: string;
  /** The labels of the Roles it holds beside Serving, in table order */
  holding: string[];
  /** This Worker's name */
  name: string;
  /** How many torrents it seeds */
  seeding: number;
  /** Since when this Worker can't reach its Controller; absent while it can */
  since?: string;
}
