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
var _HTMLScriptElementNamedNodeMap_scriptLoader;
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
class HTMLScriptElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param scriptLoader Script loader.
     */
    constructor(ownerElement, scriptLoader) {
        super(ownerElement);
        _HTMLScriptElementNamedNodeMap_scriptLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLScriptElementNamedNodeMap_scriptLoader, scriptLoader, "f");
    }
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (item[PropertySymbol.name] === 'src' &&
            item[PropertySymbol.value] !== null &&
            this[PropertySymbol.ownerElement][PropertySymbol.isConnected]) {
            __classPrivateFieldGet(this, _HTMLScriptElementNamedNodeMap_scriptLoader, "f").loadScript(item[PropertySymbol.value]);
        }
        return replacedItem || null;
    }
}
_HTMLScriptElementNamedNodeMap_scriptLoader = new WeakMap(), PropertySymbol.ownerElement;
export default HTMLScriptElementNamedNodeMap;
//# sourceMappingURL=HTMLScriptElementNamedNodeMap.js.map