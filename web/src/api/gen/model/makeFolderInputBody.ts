export interface MakeFolderInputBody {
  /**
   * The new folder's name
   * @minLength 1
   * @maxLength 255
   */
  name: string;
  /**
   * An existing folder on the Node
   * @minLength 1
   * @maxLength 4096
   */
  parent: string;
}
