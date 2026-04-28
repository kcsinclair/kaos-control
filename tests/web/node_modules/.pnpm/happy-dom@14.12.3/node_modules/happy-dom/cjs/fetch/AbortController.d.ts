import AbortSignal from './AbortSignal.cjs';
/**
 * AbortController.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortController
 */
export default class AbortController {
    readonly signal: AbortSignal;
    /**
     * Constructor.
     */
    constructor();
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    abort(reason?: Error): void;
}
//# sourceMappingURL=AbortController.d.ts.map