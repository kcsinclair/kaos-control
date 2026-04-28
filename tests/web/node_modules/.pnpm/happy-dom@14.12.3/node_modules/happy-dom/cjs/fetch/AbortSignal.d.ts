import EventTarget from '../event/EventTarget.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import Event from '../event/Event.cjs';
/**
 * AbortSignal.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
 */
export default class AbortSignal extends EventTarget {
    readonly aborted: boolean;
    readonly reason: Error | null;
    onabort: ((this: AbortSignal, event: Event) => void) | null;
    /**
     * Return a default description for the AbortSignal class.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    [PropertySymbol.abort](reason?: Error): void;
    /**
     * Throws an "AbortError" if the signal has been aborted.
     */
    throwIfAborted(): void;
    /**
     * Returns an AbortSignal instance that has been set as aborted.
     *
     * @param [reason] Reason.
     * @returns AbortSignal instance.
     */
    static abort(reason?: Error): AbortSignal;
}
//# sourceMappingURL=AbortSignal.d.ts.map