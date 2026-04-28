/**
 * Listens for uncaught exceptions coming from Happy DOM on the running Node process and dispatches error events on the Window instance.
 */
class BrowserFrameExceptionObserver {
    constructor() {
        this.browserFrame = null;
        this.uncaughtExceptionListener = null;
        this.uncaughtRejectionListener = null;
    }
    /**
     * Observes the Node process for uncaught exceptions.
     *
     * @param browserFrame Browser frame.
     */
    observe(browserFrame) {
        if (this.browserFrame) {
            throw new Error('Already observing.');
        }
        this.browserFrame = browserFrame;
        this.constructor.listenerCount++;
        this.uncaughtExceptionListener = (error, origin) => {
            if (origin === 'unhandledRejection') {
                return;
            }
            if (!this.browserFrame.window) {
                throw new Error('Browser frame was not closed correctly. Window is undefined on browser frame, but exception observer is still watching.');
            }
            if (error instanceof this.browserFrame.window.Error ||
                error instanceof this.browserFrame.window.DOMException) {
                this.browserFrame.window.console.error(error);
                this.browserFrame.window.dispatchEvent(new this.browserFrame.window.ErrorEvent('error', { error, message: error.message }));
            }
            else if (process.listenerCount('uncaughtException') ===
                this.constructor.listenerCount) {
                // eslint-disable-next-line no-console
                console.error(error);
                // Exit if there are no other listeners handling the error.
                process.exit(1);
            }
        };
        // The "uncaughtException" event is not always triggered for unhandled rejections.
        // Therefore we want to use the "unhandledRejection" event as well.
        this.uncaughtRejectionListener = (error) => {
            if (!this.browserFrame.window) {
                throw new Error('Browser frame was not closed correctly. Window is undefined on browser frame, but exception observer is still watching.');
            }
            if (error instanceof this.browserFrame.window.Error ||
                error instanceof this.browserFrame.window.DOMException) {
                this.browserFrame.window.console.error(error);
                this.browserFrame.window.dispatchEvent(new this.browserFrame.window.ErrorEvent('error', { error, message: error.message }));
            }
            else if (process.listenerCount('unhandledRejection') ===
                this.constructor.listenerCount) {
                // eslint-disable-next-line no-console
                console.error(error);
                // Exit if there are no other listeners handling the error.
                process.exit(1);
            }
        };
        process.on('uncaughtException', this.uncaughtExceptionListener);
        process.on('unhandledRejection', this.uncaughtRejectionListener);
    }
    /**
     * Disconnects observer.
     */
    disconnect() {
        if (!this.browserFrame) {
            return;
        }
        this.constructor.listenerCount--;
        process.off('uncaughtException', this.uncaughtExceptionListener);
        process.off('unhandledRejection', this.uncaughtRejectionListener);
        this.uncaughtExceptionListener = null;
        this.uncaughtRejectionListener = null;
        this.browserFrame = null;
    }
}
BrowserFrameExceptionObserver.listenerCount = 0;
export default BrowserFrameExceptionObserver;
//# sourceMappingURL=BrowserFrameExceptionObserver.js.map