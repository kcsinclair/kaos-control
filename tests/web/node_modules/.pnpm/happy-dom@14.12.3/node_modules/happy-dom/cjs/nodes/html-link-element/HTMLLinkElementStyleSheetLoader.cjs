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
var _HTMLLinkElementStyleSheetLoader_element, _HTMLLinkElementStyleSheetLoader_browserFrame, _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL;
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const ResourceFetch_js_1 = __importDefault(require("../../fetch/ResourceFetch.cjs"));
const CSSStyleSheet_js_1 = __importDefault(require("../../css/CSSStyleSheet.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const WindowErrorUtility_js_1 = __importDefault(require("../../window/WindowErrorUtility.cjs"));
/**
 * Helper class for getting the URL relative to a Location object.
 */
class HTMLLinkElementStyleSheetLoader {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Element.
     * @param options.browserFrame Browser frame.
     */
    constructor(options) {
        _HTMLLinkElementStyleSheetLoader_element.set(this, void 0);
        _HTMLLinkElementStyleSheetLoader_browserFrame.set(this, void 0);
        _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL.set(this, null);
        __classPrivateFieldSet(this, _HTMLLinkElementStyleSheetLoader_element, options.element, "f");
        __classPrivateFieldSet(this, _HTMLLinkElementStyleSheetLoader_browserFrame, options.browserFrame, "f");
    }
    /**
     * Returns a URL relative to the given Location object.
     *
     * @param url URL.
     * @param rel Rel.
     */
    async loadStyleSheet(url, rel) {
        const element = __classPrivateFieldGet(this, _HTMLLinkElementStyleSheetLoader_element, "f");
        const browserSettings = __classPrivateFieldGet(this, _HTMLLinkElementStyleSheetLoader_browserFrame, "f").page.context.browser.settings;
        const window = element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow];
        if (!url ||
            !rel ||
            rel.toLowerCase() !== 'stylesheet' ||
            !element[PropertySymbol.isConnected]) {
            return;
        }
        let absoluteURL;
        try {
            absoluteURL = new URL(url, window.location.href).href;
        }
        catch (error) {
            return;
        }
        if (__classPrivateFieldGet(this, _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL, "f") === absoluteURL) {
            return;
        }
        if (browserSettings.disableCSSFileLoading) {
            if (browserSettings.handleDisabledFileLoadingAsSuccess) {
                element.dispatchEvent(new Event_js_1.default('load'));
            }
            else {
                WindowErrorUtility_js_1.default.dispatchError(element, new DOMException_js_1.default(`Failed to load external stylesheet "${absoluteURL}". CSS file loading is disabled.`, DOMExceptionNameEnum_js_1.default.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch_js_1.default({
            browserFrame: __classPrivateFieldGet(this, _HTMLLinkElementStyleSheetLoader_browserFrame, "f"),
            window: window
        });
        const readyStateManager = window[PropertySymbol.readyStateManager];
        __classPrivateFieldSet(this, _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL, absoluteURL, "f");
        readyStateManager.startTask();
        let code = null;
        let error = null;
        try {
            code = await resourceFetch.fetch(absoluteURL);
        }
        catch (e) {
            error = e;
        }
        readyStateManager.endTask();
        if (error) {
            WindowErrorUtility_js_1.default.dispatchError(element, error);
        }
        else {
            const styleSheet = new CSSStyleSheet_js_1.default();
            styleSheet.replaceSync(code);
            element[PropertySymbol.sheet] = styleSheet;
            element.dispatchEvent(new Event_js_1.default('load'));
        }
    }
}
_HTMLLinkElementStyleSheetLoader_element = new WeakMap(), _HTMLLinkElementStyleSheetLoader_browserFrame = new WeakMap(), _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL = new WeakMap();
exports.default = HTMLLinkElementStyleSheetLoader;
//# sourceMappingURL=HTMLLinkElementStyleSheetLoader.cjs.map