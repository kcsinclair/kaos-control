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
var _HTMLIFrameElement_contentWindowContainer, _HTMLIFrameElement_pageLoader, _a;
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLIFrameElementNamedNodeMap from './HTMLIFrameElementNamedNodeMap.js';
import HTMLIFrameElementPageLoader from './HTMLIFrameElementPageLoader.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
/**
 * HTML Iframe Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLIFrameElement.
 */
export default class HTMLIFrameElement extends HTMLElement {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        super();
        // Events
        this.onload = null;
        this.onerror = null;
        this[_a] = null;
        // Private properties
        _HTMLIFrameElement_contentWindowContainer.set(this, {
            window: null
        });
        _HTMLIFrameElement_pageLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLIFrameElement_pageLoader, new HTMLIFrameElementPageLoader({
            element: this,
            contentWindowContainer: __classPrivateFieldGet(this, _HTMLIFrameElement_contentWindowContainer, "f"),
            browserParentFrame: browserFrame
        }), "f");
        this[PropertySymbol.attributes] = new HTMLIFrameElementNamedNodeMap(this, __classPrivateFieldGet(this, _HTMLIFrameElement_pageLoader, "f"));
    }
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get src() {
        return this.getAttribute('src') || '';
    }
    /**
     * Sets source.
     *
     * @param src Source.
     */
    set src(src) {
        this.setAttribute('src', src);
    }
    /**
     * Returns allow.
     *
     * @returns Allow.
     */
    get allow() {
        return this.getAttribute('allow') || '';
    }
    /**
     * Sets allow.
     *
     * @param allow Allow.
     */
    set allow(allow) {
        this.setAttribute('allow', allow);
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this.getAttribute('height') || '';
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this.setAttribute('height', height);
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this.getAttribute('width') || '';
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this.setAttribute('width', width);
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
    /**
     * Returns sandbox.
     *
     * @returns Sandbox.
     */
    get sandbox() {
        if (!this[PropertySymbol.sandbox]) {
            this[PropertySymbol.sandbox] = new DOMTokenList(this, 'sandbox');
        }
        return this[PropertySymbol.sandbox];
    }
    /**
     * Sets sandbox.
     */
    set sandbox(sandbox) {
        this.setAttribute('sandbox', sandbox);
    }
    /**
     * Returns srcdoc.
     *
     * @returns Srcdoc.
     */
    get srcdoc() {
        return this.getAttribute('srcdoc') || '';
    }
    /**
     * Sets srcdoc.
     *
     * @param srcdoc Srcdoc.
     */
    set srcdoc(srcdoc) {
        this.setAttribute('srcdoc', srcdoc);
    }
    /**
     * Returns referrer policy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerpolicy') || '';
    }
    /**
     * Sets referrer policy.
     *
     * @param referrerPolicy Referrer policy.
     */
    set referrerPolicy(referrerPolicy) {
        this.setAttribute('referrerpolicy', referrerPolicy);
    }
    /**
     * Returns content document.
     *
     * @returns Content document.
     */
    get contentDocument() {
        return __classPrivateFieldGet(this, _HTMLIFrameElement_contentWindowContainer, "f").window?.document ?? null;
    }
    /**
     * Returns content window.
     *
     * @returns Content window.
     */
    get contentWindow() {
        return __classPrivateFieldGet(this, _HTMLIFrameElement_contentWindowContainer, "f").window;
    }
    /**
     * @override
     */
    [(_HTMLIFrameElement_contentWindowContainer = new WeakMap(), _HTMLIFrameElement_pageLoader = new WeakMap(), PropertySymbol.attributes, _a = PropertySymbol.sandbox, PropertySymbol.connectToNode)](parentNode = null) {
        const isConnected = this[PropertySymbol.isConnected];
        const isParentConnected = parentNode ? parentNode[PropertySymbol.isConnected] : false;
        super[PropertySymbol.connectToNode](parentNode);
        if (isConnected !== isParentConnected) {
            if (isParentConnected) {
                __classPrivateFieldGet(this, _HTMLIFrameElement_pageLoader, "f").loadPage();
            }
            else {
                __classPrivateFieldGet(this, _HTMLIFrameElement_pageLoader, "f").unloadPage();
            }
        }
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLIFrameElement.js.map