// PROTOTYPE · Variant F · Topology with live traffic. D's diagram, plus every running job drawn on the links its bytes cross.
import { topology } from './topology.js';

export const { name, render, modalHtml, actions } = topology({ id: 'F', name: 'Topology + traffic', traffic: true });
