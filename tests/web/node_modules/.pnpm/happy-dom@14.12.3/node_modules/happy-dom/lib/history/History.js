var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _History_scrollRestoration;
import HistoryScrollRestorationEnum from './HistoryScrollRestorationEnum.js';
/**
 * History API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/History.
 */
class History {
    constructor() {
        this.length = 0;
        this.state = null;
        _History_scrollRestoration.set(this, HistoryScrollRestorationEnum.auto);
    }
    /**
     * Returns scroll restoration.
     *
     * @returns Sroll restoration.
     */
    get scrollRestoration() {
        return __classPrivateFieldGet(this, _History_scrollRestoration, "f");
    }
    /**
     * Sets scroll restoration.
     *
     * @param scrollRestoration Sroll restoration.
     */
    set scrollRestoration(scrollRestoration) {
        __classPrivateFieldSet(this, _History_scrollRestoration, HistoryScrollRestorationEnum[scrollRestoration]
            ? scrollRestoration
            : __classPrivateFieldGet(this, _History_scrollRestoration, "f"), "f");
    }
    /**
     * Goes to the previous page in session history.
     */
    back() {
        // Do nothing.
    }
    /**
     * Goes to the next page in session history.
     */
    forward() {
        // Do nothing.
    }
    /**
     * Load a specific page from the session history.
     *
     * @param delta Delta.
     * @param _delta
     */
    go(_delta) {
        // Do nothing.
    }
    /**
     * Pushes the given data onto the session history stack.
     *
     * @param state State.
     * @param title Title.
     * @param [url] URL.
     * @param _state
     * @param _title
     * @param _url
     */
    pushState(_state, _title, _url) {
        // Do nothing.
    }
    /**
     * This method modifies the current history entry, replacing it with a new state.
     *
     * @param state State.
     * @param title Title.
     * @param [url] URL.
     * @param _state
     * @param _title
     * @param _url
     */
    replaceState(_state, _title, _url) {
        // Do nothing.
    }
}
_History_scrollRestoration = new WeakMap();
export default History;
//# sourceMappingURL=History.js.map