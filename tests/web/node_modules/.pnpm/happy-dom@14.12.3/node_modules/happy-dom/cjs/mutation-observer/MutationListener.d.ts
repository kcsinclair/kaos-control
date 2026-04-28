import IMutationObserverInit from './IMutationObserverInit.cjs';
import MutationObserver from './MutationObserver.cjs';
import MutationRecord from './MutationRecord.cjs';
import Node from '../nodes/node/Node.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * Mutation Observer Listener.
 */
export default class MutationListener {
    #private;
    readonly target: Node;
    options: IMutationObserverInit;
    /**
     * Constructor.
     *
     * @param init Options.
     * @param init.window Window.
     * @param init.options Options.
     * @param init.target Target.
     * @param init.observer Observer.
     * @param init.callback Callback.
     */
    constructor(init: {
        window: BrowserWindow;
        options: IMutationObserverInit;
        target: Node;
        observer: MutationObserver;
        callback: (record: MutationRecord[], observer: MutationObserver) => void;
    });
    /**
     * Reports mutations.
     *
     * @param record Record.
     */
    report(record: MutationRecord): void;
    /**
     * Destroys the listener.
     */
    takeRecords(): MutationRecord[];
    /**
     * Destroys the listener.
     */
    destroy(): void;
}
//# sourceMappingURL=MutationListener.d.ts.map