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
var _HTMLLinkElementNamedNodeMap_styleSheetLoader;
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLLinkElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param stylesheetLoader Stylesheet loader.
     * @param styleSheetLoader
     */
    constructor(ownerElement, styleSheetLoader) {
        super(ownerElement);
        _HTMLLinkElementNamedNodeMap_styleSheetLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLLinkElementNamedNodeMap_styleSheetLoader, styleSheetLoader, "f");
    }
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (item[PropertySymbol.name] === 'rel' &&
            this[PropertySymbol.ownerElement][PropertySymbol.relList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.relList][PropertySymbol.updateIndices]();
        }
        if (item[PropertySymbol.name] === 'rel') {
            __classPrivateFieldGet(this, _HTMLLinkElementNamedNodeMap_styleSheetLoader, "f").loadStyleSheet(this[PropertySymbol.ownerElement].getAttribute('href'), item[PropertySymbol.value]);
        }
        else if (item[PropertySymbol.name] === 'href') {
            __classPrivateFieldGet(this, _HTMLLinkElementNamedNodeMap_styleSheetLoader, "f").loadStyleSheet(item[PropertySymbol.value], this[PropertySymbol.ownerElement].getAttribute('rel'));
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(_HTMLLinkElementNamedNodeMap_styleSheetLoader = new WeakMap(), PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem &&
            removedItem[PropertySymbol.name] === 'rel' &&
            this[PropertySymbol.ownerElement][PropertySymbol.relList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.relList][PropertySymbol.updateIndices]();
        }
        return removedItem;
    }
}
//# sourceMappingURL=HTMLLinkElementNamedNodeMap.js.map