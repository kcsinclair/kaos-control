"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _HTMLLinkElement_styleSheetLoader, _a, _b, _c;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const DOMTokenList_js_1 = __importDefault(require("../../dom-token-list/DOMTokenList.cjs"));
const HTMLLinkElementNamedNodeMap_js_1 = __importDefault(require("./HTMLLinkElementNamedNodeMap.cjs"));
const HTMLLinkElementStyleSheetLoader_js_1 = __importDefault(require("./HTMLLinkElementStyleSheetLoader.cjs"));
/**
 * HTML Link Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLinkElement.
 */
class HTMLLinkElement extends HTMLElement_js_1.default {
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
        __classPrivateFieldSet(this, _HTMLLinkElement_styleSheetLoader, new HTMLLinkElementStyleSheetLoader_js_1.default({
            element: this,
            browserFrame
        }), "f");
        this[PropertySymbol.attributes] = new HTMLLinkElementNamedNodeMap_js_1.default(this, __classPrivateFieldGet(this, _HTMLLinkElement_styleSheetLoader, "f"));
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
            this[PropertySymbol.relList] = new DOMTokenList_js_1.default(this, 'rel');
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
exports.default = HTMLLinkElement;
//# sourceMappingURL=HTMLLinkElement.cjs.map