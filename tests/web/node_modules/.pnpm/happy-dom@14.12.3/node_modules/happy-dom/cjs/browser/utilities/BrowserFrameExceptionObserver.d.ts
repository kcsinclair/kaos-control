import IBrowserFrame from '../types/IBrowserFrame.cjs';
/**
 * Listens for uncaught exceptions coming from Happy DOM on the running Node process and dispatches error events on the Window instance.
 */
export default class BrowserFrameExceptionObserver {
    private static listenerCount;
    private browserFrame;
    private uncaughtExceptionListener;
    private uncaughtRejectionListener;
    /**
     * Observes the Node process for uncaught exceptions.
     *
     * @param browserFrame Browser frame.
     */
    observe(browserFrame: IBrowserFrame): void;
    /**
     * Disconnects observer.
     */
    disconnect(): void;
}
//# sourceMappingURL=BrowserFrameExceptionObserver.d.ts.map