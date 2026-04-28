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
var _HTMLScriptElement_scriptLoader, _a;
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLScriptElementNamedNodeMap_js_1 = __importDefault(require("./HTMLScriptElementNamedNodeMap.cjs"));
const WindowErrorUtility_js_1 = __importDefault(require("../../window/WindowErrorUtility.cjs"));
const WindowBrowserSettingsReader_js_1 = __importDefault(require("../../window/WindowBrowserSettingsReader.cjs"));
const HTMLScriptElementScriptLoader_js_1 = __importDefault(require("./HTMLScriptElementScriptLoader.cjs"));
const BrowserErrorCaptureEnum_js_1 = __importDefault(require("../../browser/enums/BrowserErrorCaptureEnum.cjs"));
/**
 * HTML Script Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLScriptElement.
 */
class HTMLScriptElement extends HTMLElement_js_1.default {
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
        this[_a] = true;
        // Private properties
        _HTMLScriptElement_scriptLoader.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLScriptElement_scriptLoader, new HTMLScriptElementScriptLoader_js_1.default({
            element: this,
            browserFrame
        }), "f");
        this[PropertySymbol.attributes] = new HTMLScriptElementNamedNodeMap_js_1.default(this, __classPrivateFieldGet(this, _HTMLScriptElement_scriptLoader, "f"));
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
     * Returns source.
     *
     * @returns Source.
     */
    get src() {
        if (!this.hasAttribute('src')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('src');
        }
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
     * Returns charset.
     *
     * @returns Charset.
     */
    get charset() {
        return this.getAttribute('charset') || '';
    }
    /**
     * Sets charset.
     *
     * @param charset Charset.
     */
    set charset(charset) {
        this.setAttribute('charset', charset);
    }
    /**
     * Returns lang.
     *
     * @returns Lang.
     */
    get lang() {
        return this.getAttribute('lang') || '';
    }
    /**
     * Sets lang.
     *
     * @param lang Lang.
     */
    set lang(lang) {
        this.setAttribute('lang', lang);
    }
    /**
     * Returns async.
     *
     * @returns Async.
     */
    get async() {
        return this.getAttribute('async') !== null;
    }
    /**
     * Sets async.
     *
     * @param async Async.
     */
    set async(async) {
        if (!async) {
            this.removeAttribute('async');
        }
        else {
            this.setAttribute('async', '');
        }
    }
    /**
     * Returns defer.
     *
     * @returns Defer.
     */
    get defer() {
        return this.getAttribute('defer') !== null;
    }
    /**
     * Sets defer.
     *
     * @param defer Defer.
     */
    set defer(defer) {
        if (!defer) {
            this.removeAttribute('defer');
        }
        else {
            this.setAttribute('defer', '');
        }
    }
    /**
     * Returns text.
     *
     * @returns Text.
     */
    get text() {
        return this.textContent;
    }
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text) {
        this.textContent = text;
    }
    /**
     * @override
     */
    [(_HTMLScriptElement_scriptLoader = new WeakMap(), PropertySymbol.attributes, _a = PropertySymbol.evaluateScript, PropertySymbol.cloneNode)](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        const isConnected = this[PropertySymbol.isConnected];
        const isParentConnected = parentNode ? parentNode[PropertySymbol.isConnected] : false;
        const browserSettings = WindowBrowserSettingsReader_js_1.default.getSettings(this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow]);
        super[PropertySymbol.connectToNode](parentNode);
        if (isParentConnected &&
            isConnected !== isParentConnected &&
            this[PropertySymbol.evaluateScript]) {
            const src = this.getAttribute('src');
            if (src !== null) {
                __classPrivateFieldGet(this, _HTMLScriptElement_scriptLoader, "f").loadScript(src);
            }
            else if (!browserSettings.disableJavaScriptEvaluation) {
                const textContent = this.textContent;
                const type = this.getAttribute('type');
                if (textContent &&
                    (type === null ||
                        type === 'application/x-ecmascript' ||
                        type === 'application/x-javascript' ||
                        type.startsWith('text/javascript'))) {
                    this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = this;
                    const code = `//# sourceURL=${this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].location.href}\n` + textContent;
                    if (browserSettings.disableErrorCapturing ||
                        browserSettings.errorCapture !== BrowserErrorCaptureEnum_js_1.default.tryAndCatch) {
                        this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code);
                    }
                    else {
                        WindowErrorUtility_js_1.default.captureError(this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow], () => this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code));
                    }
                    this[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = null;
                }
            }
        }
    }
}
exports.default = HTMLScriptElement;
//# sourceMappingURL=HTMLScriptElement.cjs.map