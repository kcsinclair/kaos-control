import Node from '../nodes/node/Node.js';
import IMutationObserverInit from './IMutationObserverInit.js';
import MutationRecord from './MutationRecord.js';
/**
 * The MutationObserver interface provides the ability to watch for changes being made to the DOM tree.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver
 */
export default class MutationObserver {
    #private;
    /**
     * Constructor.
     *
     * @param callback Callback.
     */
    constructor(callback: (records: MutationRecord[], observer: MutationObserver) => void);
    /**
     * Starts observing.
     *
     * @param target Target.
     * @param options Options.
     */
    observe(target: Node, options: IMutationObserverInit): void;
    /**
     * Disconnects.
     */
    disconnect(): void;
    /**
     * Returns a list of all matching DOM changes that have been detected but not yet processed by the observer's callback function, leaving the mutation queue empty.
     *
     * @returns Records.
     */
    takeRecords(): MutationRecord[];
}
//# sourceMappingURL=MutationObserver.d.ts.map