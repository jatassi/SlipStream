export interface GrabsChanged {
  grabId: string;
  /** True when only the download's progress moved */
  progress: boolean;
  workId: string;
}
