// PROTOTYPE stub, replaced by the inspector variant.
import { cells, counts } from '../model.js';
export const wide = ({ work, desc }) => `<pre>${JSON.stringify(counts(cells(work, desc)))}</pre>`;
export const phone = wide;
