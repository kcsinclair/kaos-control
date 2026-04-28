var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _MediaQueryList_ownerWindow, _MediaQueryList_items, _MediaQueryList_media, _MediaQueryList_rootFontSize;
import EventTarget from '../event/EventTarget.js';
import * as PropertySymbol from '../PropertySymbol.js';
import MediaQueryListEvent from '../event/events/MediaQueryListEvent.js';
import MediaQueryParser from './MediaQueryParser.js';
/**
 * Media Query List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/MediaQueryList.
 */
class MediaQueryList extends EventTarget {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.ownerWindow Owner window.
     * @param options.media Media.
     * @param [options.rootFontSize] Root font size.
     */
    constructor(options) {
        super();
        this.onchange = null;
        _MediaQueryList_ownerWindow.set(this, void 0);
        _MediaQueryList_items.set(this, null);
        _MediaQueryList_media.set(this, void 0);
        _MediaQueryList_rootFontSize.set(this, null);
        __classPrivateFieldSet(this, _MediaQueryList_ownerWindow, options.ownerWindow, "f");
        __classPrivateFieldSet(this, _MediaQueryList_media, options.media, "f");
        __classPrivateFieldSet(this, _MediaQueryList_rootFontSize, options.rootFontSize || null, "f");
    }
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        __classPrivateFieldSet(this, _MediaQueryList_items, __classPrivateFieldGet(this, _MediaQueryList_items, "f") ||
            MediaQueryParser.parse({
                ownerWindow: __classPrivateFieldGet(this, _MediaQueryList_ownerWindow, "f"),
                mediaQuery: __classPrivateFieldGet(this, _MediaQueryList_media, "f"),
                rootFontSize: __classPrivateFieldGet(this, _MediaQueryList_rootFontSize, "f")
            }), "f");
        return __classPrivateFieldGet(this, _MediaQueryList_items, "f").map((item) => item.toString()).join(', ');
    }
    /**
     * Returns "true" if the document matches.
     *
     * @returns Matches.
     */
    get matches() {
        __classPrivateFieldSet(this, _MediaQueryList_items, __classPrivateFieldGet(this, _MediaQueryList_items, "f") ||
            MediaQueryParser.parse({
                ownerWindow: __classPrivateFieldGet(this, _MediaQueryList_ownerWindow, "f"),
                mediaQuery: __classPrivateFieldGet(this, _MediaQueryList_media, "f"),
                rootFontSize: __classPrivateFieldGet(this, _MediaQueryList_rootFontSize, "f")
            }), "f");
        for (const item of __classPrivateFieldGet(this, _MediaQueryList_items, "f")) {
            if (!item.matches()) {
                return false;
            }
        }
        return true;
    }
    /**
     * Adds a listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    addListener(callback) {
        this.addEventListener('change', callback);
    }
    /**
     * Removes listener.
     *
     * @deprecated
     * @param callback Callback.
     */
    removeListener(callback) {
        this.removeEventListener('change', callback);
    }
    /**
     * @override
     */
    addEventListener(type, listener) {
        super.addEventListener(type, listener);
        if (type === 'change') {
            let matchesState = false;
            const resizeListener = () => {
                const matches = this.matches;
                if (matches !== matchesState) {
                    matchesState = matches;
                    this.dispatchEvent(new MediaQueryListEvent('change', { matches, media: this.media }));
                }
            };
            listener[PropertySymbol.windowResizeListener] = resizeListener;
            __classPrivateFieldGet(this, _MediaQueryList_ownerWindow, "f").addEventListener('resize', resizeListener);
        }
    }
    /**
     * @override
     */
    removeEventListener(type, listener) {
        super.removeEventListener(type, listener);
        if (type === 'change' && listener[PropertySymbol.windowResizeListener]) {
            __classPrivateFieldGet(this, _MediaQueryList_ownerWindow, "f").removeEventListener('resize', listener[PropertySymbol.windowResizeListener]);
        }
    }
}
_MediaQueryList_ownerWindow = new WeakMap(), _MediaQueryList_items = new WeakMap(), _MediaQueryList_media = new WeakMap(), _MediaQueryList_rootFontSize = new WeakMap();
export default MediaQueryList;
//# sourceMappingURL=MediaQueryList.js.map