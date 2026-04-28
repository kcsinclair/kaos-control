import AbortSignal from './AbortSignal.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * AbortController.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortController
 */
export default class AbortController {
    /**
     * Constructor.
     */
    constructor() {
        this.signal = new AbortSignal();
        this.signal = new AbortSignal();
    }
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    abort(reason) {
        this.signal[PropertySymbol.abort](reason);
    }
}
//# sourceMappingURL=AbortController.js.map