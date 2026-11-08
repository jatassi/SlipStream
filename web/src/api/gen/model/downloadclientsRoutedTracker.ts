export interface DownloadclientsRoutedTracker {
  id: string;
  name: string;
  /** An Admin picked this client for the Tracker's Grabs from some home Node; false while they come here by default */
  stored: boolean;
}
