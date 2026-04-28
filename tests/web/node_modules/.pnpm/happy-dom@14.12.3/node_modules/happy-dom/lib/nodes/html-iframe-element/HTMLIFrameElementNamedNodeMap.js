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
var _HTMLIFrameElementNamedNodeMap_instances, _HTMLIFrameElementNamedNodeMap_pageLoader, _HTMLIFrameElementNamedNodeMap_validateSandboxFlags;
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
const SANDBOX_FLAGS = [
    'allow-downloads',
    'allow-forms',
    'allow-modals',
    'allow-orientation-lock',
    'allow-pointer-lock',
    'allow-popups',
    'allow-popups-to-escape-sandbox',
    'allow-presentation',
    'allow-same-origin',
    'allow-scripts',
    'allow-top-navigation',
    'allow-top-navigation-by-user-activation',
    'allow-top-navigation-to-custom-protocols'
];
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
class HTMLIFrameElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param pageLoader
     */
    constructor(ownerElement, pageLoader) {
        super(ownerElement);
        _HTMLIFrameElementNamedNodeMap_instances.add(this);
        _HTMLIFrameElementNamedNodeMap_pageLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLIFrameElementNamedNodeMap_pageLoader, pageLoader, "f");
    }
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedAttribute = super.setNamedItem(item);
        if (item[PropertySymbol.name] === 'srcdoc') {
            __classPrivateFieldGet(this, _HTMLIFrameElementNamedNodeMap_pageLoader, "f").loadPage();
        }
        // If the src attribute and the srcdoc attribute are both specified together, the srcdoc attribute takes priority.
        if (item[PropertySymbol.name] === 'src' &&
            this[PropertySymbol.ownerElement][PropertySymbol.attributes]['srcdoc']?.value === undefined &&
            item[PropertySymbol.value] &&
            item[PropertySymbol.value] !== replacedAttribute?.[PropertySymbol.value]) {
            __classPrivateFieldGet(this, _HTMLIFrameElementNamedNodeMap_pageLoader, "f").loadPage();
        }
        if (item[PropertySymbol.name] === 'sandbox') {
            if (!this[PropertySymbol.ownerElement][PropertySymbol.sandbox]) {
                this[PropertySymbol.ownerElement][PropertySymbol.sandbox] = new DOMTokenList(this[PropertySymbol.ownerElement], 'sandbox');
            }
            else {
                this[PropertySymbol.ownerElement][PropertySymbol.sandbox][PropertySymbol.updateIndices]();
            }
            __classPrivateFieldGet(this, _HTMLIFrameElementNamedNodeMap_instances, "m", _HTMLIFrameElementNamedNodeMap_validateSandboxFlags).call(this);
        }
        return replacedAttribute || null;
    }
    /**
     * @override
     */
    [(_HTMLIFrameElementNamedNodeMap_pageLoader = new WeakMap(), _HTMLIFrameElementNamedNodeMap_instances = new WeakSet(), PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem &&
            (removedItem[PropertySymbol.name] === 'srcdoc' || removedItem[PropertySymbol.name] === 'src')) {
            __classPrivateFieldGet(this, _HTMLIFrameElementNamedNodeMap_pageLoader, "f").loadPage();
        }
        return removedItem;
    }
}
_HTMLIFrameElementNamedNodeMap_validateSandboxFlags = function _HTMLIFrameElementNamedNodeMap_validateSandboxFlags() {
    const window = this[PropertySymbol.ownerElement][PropertySymbol.ownerDocument][PropertySymbol.ownerWindow];
    const values = this[PropertySymbol.ownerElement][PropertySymbol.sandbox].values();
    const invalidFlags = [];
    for (const token of values) {
        if (!SANDBOX_FLAGS.includes(token)) {
            invalidFlags.push(token);
        }
    }
    if (invalidFlags.length === 1) {
        window.console.error(`Error while parsing the 'sandbox' attribute: '${invalidFlags[0]}' is an invalid sandbox flag.`);
    }
    else if (invalidFlags.length > 1) {
        window.console.error(`Error while parsing the 'sandbox' attribute: '${invalidFlags.join(`', '`)}' are invalid sandbox flags.`);
    }
};
export default HTMLIFrameElementNamedNodeMap;
//# sourceMappingURL=HTMLIFrameElementNamedNodeMap.js.map