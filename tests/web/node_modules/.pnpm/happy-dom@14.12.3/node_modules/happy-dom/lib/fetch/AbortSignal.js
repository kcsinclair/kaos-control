import EventTarget from '../event/EventTarget.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Event from '../event/Event.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import DOMException from '../exception/DOMException.js';
/**
 * AbortSignal.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
 */
export default class AbortSignal extends EventTarget {
    constructor() {
        super(...arguments);
        this.aborted = false;
        this.reason = null;
        this.onabort = null;
    }
    /**
     * Return a default description for the AbortSignal class.
     */
    get [Symbol.toStringTag]() {
        return 'AbortSignal';
    }
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    [PropertySymbol.abort](reason) {
        if (this.aborted) {
            return;
        }
        this.reason =
            reason ||
                new DOMException('signal is aborted without reason', DOMExceptionNameEnum.abortError);
        this.aborted = true;
        this.dispatchEvent(new Event('abort'));
    }
    /**
     * Throws an "AbortError" if the signal has been aborted.
     */
    throwIfAborted() {
        if (this.aborted) {
            throw this.reason;
        }
    }
    /**
     * Returns an AbortSignal instance that has been set as aborted.
     *
     * @param [reason] Reason.
     * @returns AbortSignal instance.
     */
    static abort(reason) {
        const signal = new AbortSignal();
        signal.reason =
            reason ||
                new DOMException('signal is aborted without reason', DOMExceptionNameEnum.abortError);
        signal.aborted = true;
        return signal;
    }
}
//# sourceMappingURL=AbortSignal.js.map