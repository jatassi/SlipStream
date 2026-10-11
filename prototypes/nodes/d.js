// PROTOTYPE · Variant D · Topology canvas. Structure first: who links to whom and which Role reaches which service.
import { topology } from './topology.js';

export const { name, render, modalHtml, actions } = topology({ id: 'D', name: 'Topology', traffic: false });
