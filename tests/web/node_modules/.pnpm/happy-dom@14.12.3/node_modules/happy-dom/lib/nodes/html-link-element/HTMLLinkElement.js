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
var _HTMLLinkElement_styleSheetLoader, _a, _b, _c;
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
import HTMLLinkElementNamedNodeMap from './HTMLLinkElementNamedNodeMap.js';
import HTMLLinkElementStyleSheetLoader from './HTMLLinkElementStyleSheetLoader.js';
/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
export default class HTMLLinkElement extends HTMLElement {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        super();
        // Events
        this.onerror = null;
        this.onload = null;
        this[_a] = null;
        this[_b] = true;
        this[_c] = null;
        _HTMLLinkElement_styleSheetLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLLinkElement_styleSheetLoader, new HTMLLinkElementStyleSheetLoader({
            element: this,
            browserFrame
        }), "f");
        this[PropertySymbol.attributes] = new HTMLLinkElementNamedNodeMap(this, __classPrivateFieldGet(this, _HTMLLinkElement_styleSheetLoader, "f"));
    }
    /**
     * Returns sheet.
     */
    get sheet() {
        return this[PropertySymbol.sheet];
    }
    /**
     * Returns rel list.
     *
     * @returns Rel list.
     */
    get relList() {
        if (!this[PropertySymbol.relList]) {
            this[PropertySymbol.relList] = new DOMTokenList(this, 'rel');
        }
        return this[PropertySymbol.relList];
    }
    /**
     * Returns as.
     *
     * @returns As.
     */
    get as() {
        return this.getAttribute('as') || '';
    }
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set as(as) {
        this.setAttribute('as', as);
    }
    /**
     * Returns crossOrigin.
     *
     * @returns CrossOrigin.
     */
    get crossOrigin() {
        return this.getAttribute('crossorigin') || '';
    }
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set crossOrigin(crossOrigin) {
        this.setAttribute('crossorigin', crossOrigin);
    }
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this.hasAttribute('href')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('href'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('href');
        }
    }
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href) {
        this.setAttribute('href', href);
    }
    /**
     * Returns hreflang.
     *
     * @returns Hreflang.
     */
    get hreflang() {
        return this.getAttribute('hreflang') || '';
    }
    /**
     * Sets hreflang.
     *
     * @param hreflang Hreflang.
     */
    set hreflang(hreflang) {
        this.setAttribute('hreflang', hreflang);
    }
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || '';
    }
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media) {
        this.setAttribute('media', media);
    }
    /**
     * Returns referrerPolicy.
     *
     * @returns ReferrerPolicy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerPolicy') || '';
    }
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy ReferrerPolicy.
     */
    set referrerPolicy(referrerPolicy) {
        this.setAttribute('referrerPolicy', referrerPolicy);
    }
    /**
     * Returns rel.
     *
     * @returns Rel.
     */
    get rel() {
        return this.getAttribute('rel') || '';
    }
    /**
     * Sets rel.
     *
     * @param rel Rel.
     */
    set rel(rel) {
        this.setAttribute('rel', rel);
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
    /**
     * @override
     */
    [(_HTMLLinkElement_styleSheetLoader = new WeakMap(), PropertySymbol.attributes, _a = PropertySymbol.sheet, _b = PropertySymbol.evaluateCSS, _c = PropertySymbol.relList, PropertySymbol.connectToNode)](parentNode = null) {
        const isConnected = this[PropertySymbol.isConnected];
        const isParentConnected = parentNode ? parentNode[PropertySymbol.isConnected] : false;
        super[PropertySymbol.connectToNode](parentNode);
        if (isParentConnected &&
            isConnected !== isParentConnected &&
            this[PropertySymbol.evaluateCSS]) {
            __classPrivateFieldGet(this, _HTMLLinkElement_styleSheetLoader, "f").loadStyleSheet(this.getAttribute('href'), this.getAttribute('rel'));
        }
    }
}
//# sourceMappingURL=HTMLLinkElement.js.map