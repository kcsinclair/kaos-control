// Layout configuration registry for Cytoscape 2D graph.
// Adding a new layout requires only appending an entry here.

export interface LayoutConfig {
  /** Unique key used as the store's activeLayout value */
  key: string
  /** Human-readable name shown in the selector */
  label: string
  /** Cytoscape layout name passed to cy.layout() */
  cyName: string
  /** Default options merged into the layout call */
  options: Record<string, unknown>
  /** Optional async import that registers a Cytoscape plugin */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  plugin?: () => Promise<any>
}

export const LAYOUT_CONFIGS: Record<string, LayoutConfig> = {
  fcose: {
    key: 'fcose',
    label: 'fCoSE (Force-Directed)',
    cyName: 'fcose',
    options: {
      name: 'fcose',
      quality: 'proof',
      randomize: true,
      animate: true,
      animationDuration: 400,
      nodeSeparation: 120,
      idealEdgeLength: () => 80,
    },
    plugin: () => import('cytoscape-fcose'),
  },
  breadthfirst: {
    key: 'breadthfirst',
    label: 'Breadth-First',
    cyName: 'breadthfirst',
    options: {
      name: 'breadthfirst',
      // directed: true is the default; overridden by the directed toggle at runtime
      directed: true,
      padding: 40,
      spacingFactor: 1.6,
      avoidOverlap: true,
      animate: true,
      animationDuration: 400,
    },
  },
  concentric: {
    key: 'concentric',
    label: 'Concentric',
    cyName: 'concentric',
    options: {
      name: 'concentric',
      animate: true,
      animationDuration: 400,
      padding: 30,
      avoidOverlap: true,
    },
  },
  circle: {
    key: 'circle',
    label: 'Circle',
    cyName: 'circle',
    options: {
      name: 'circle',
      animate: true,
      animationDuration: 400,
      padding: 30,
      avoidOverlap: true,
    },
  },
  dagre: {
    key: 'dagre',
    label: 'Dagre (DAG)',
    cyName: 'dagre',
    options: {
      name: 'dagre',
      animate: true,
      animationDuration: 400,
      padding: 30,
      // directed: true is the default; overridden by the directed toggle at runtime
      directed: true,
    },
    // plugin added in Milestone 5 when cytoscape-dagre is installed
  },
}
