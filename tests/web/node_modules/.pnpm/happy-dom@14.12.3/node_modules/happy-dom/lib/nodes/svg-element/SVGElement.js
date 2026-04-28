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
var _SVGElement_dataset, _a, _b;
import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import HTMLElementUtility from '../html-element/HTMLElementUtility.js';
import SVGElementNamedNodeMap from './SVGElementNamedNodeMap.js';
import DatasetFactory from '../element/DatasetFactory.js';
/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
class SVGElement extends Element {
    constructor() {
        super(...arguments);
        // Events
        this.onabort = null;
        this.onerror = null;
        this.onload = null;
        this.onresize = null;
        this.onscroll = null;
        this.onunload = null;
        // Internal properties
        this[_a] = new SVGElementNamedNodeMap(this);
        this[_b] = null;
        // Private properties
        _SVGElement_dataset.set(this, null);
    }
    /**
     * Returns viewport.
     *
     * @returns SVG rect.
     */
    get viewportElement() {
        return null;
    }
    /**
     * Returns current translate.
     *
     * @returns Element.
     */
    get ownerSVGElement() {
        let parent = this[PropertySymbol.parentNode];
        while (parent) {
            if (parent[PropertySymbol.localName] === 'svg') {
                return parent;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return null;
    }
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset() {
        return (__classPrivateFieldSet(this, _SVGElement_dataset, __classPrivateFieldGet(this, _SVGElement_dataset, "f") ?? DatasetFactory.createDataset(this), "f"));
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this[PropertySymbol.style]) {
            this[PropertySymbol.style] = new CSSStyleDeclaration(this);
        }
        return this[PropertySymbol.style];
    }
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex() {
        const tabIndex = this.getAttribute('tabindex');
        return tabIndex !== null ? Number(tabIndex) : -1;
    }
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex) {
        if (tabIndex === -1) {
            this.removeAttribute('tabindex');
        }
        else {
            this.setAttribute('tabindex', String(tabIndex));
        }
    }
    /**
     * Triggers a blur event.
     */
    blur() {
        HTMLElementUtility.blur(this);
    }
    /**
     * Triggers a focus event.
     */
    focus() {
        HTMLElementUtility.focus(this);
    }
}
_SVGElement_dataset = new WeakMap(), _a = PropertySymbol.attributes, _b = PropertySymbol.style;
export default SVGElement;
//# sourceMappingURL=SVGElement.js.map