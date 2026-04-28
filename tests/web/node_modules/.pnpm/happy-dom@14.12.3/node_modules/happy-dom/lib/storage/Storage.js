var _a;
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Storage.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Storage
 */
class Storage {
    constructor() {
        this[_a] = {};
    }
    /**
     * Returns length.
     *
     * @returns Length.
     */
    get length() {
        return Object.keys(this[PropertySymbol.data]).length;
    }
    /**
     * Returns name of the nth key.
     *
     * @param index Index.
     * @returns Name.
     */
    key(index) {
        const name = Object.keys(this[PropertySymbol.data])[index];
        return name !== undefined ? name : null;
    }
    /**
     * Sets item.
     *
     * @param name Name.
     * @param item Item.
     */
    setItem(name, item) {
        this[PropertySymbol.data][name] = String(item);
    }
    /**
     * Returns item.
     *
     * @param name Name.
     * @returns Item.
     */
    getItem(name) {
        return this[PropertySymbol.data][name] !== undefined ? this[PropertySymbol.data][name] : null;
    }
    /**
     * Removes item.
     *
     * @param name Name.
     */
    removeItem(name) {
        delete this[PropertySymbol.data][name];
    }
    /**
     * Clears storage.
     */
    clear() {
        this[PropertySymbol.data] = {};
    }
}
_a = PropertySymbol.data;
export default Storage;
//# sourceMappingURL=Storage.js.map