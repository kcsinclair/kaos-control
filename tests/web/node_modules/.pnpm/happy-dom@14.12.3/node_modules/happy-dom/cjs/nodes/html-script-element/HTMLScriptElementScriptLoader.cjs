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
var _HTMLScriptElementScriptLoader_element, _HTMLScriptElementScriptLoader_browserFrame, _HTMLScriptElementScriptLoader_loadedScriptURL;
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const ResourceFetch_js_1 = __importDefault(require("../../fetch/ResourceFetch.cjs"));
const WindowErrorUtility_js_1 = __importDefault(require("../../window/WindowErrorUtility.cjs"));
const BrowserErrorCaptureEnum_js_1 = __importDefault(require("../../browser/enums/BrowserErrorCaptureEnum.cjs"));
/**
 * Helper class for getting the URL relative to a Location object.
 */
class HTMLScriptElementScriptLoader {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Element.
     * @param options.browserFrame Browser frame.
     */
    constructor(options) {
        _HTMLScriptElementScriptLoader_element.set(this, void 0);
        _HTMLScriptElementScriptLoader_browserFrame.set(this, void 0);
        _HTMLScriptElementScriptLoader_loadedScriptURL.set(this, null);
        __classPrivateFieldSet(this, _HTMLScriptElementScriptLoader_element, options.element, "f");
        __classPrivateFieldSet(this, _HTMLScriptElementScriptLoader_browserFrame, options.browserFrame, "f");
    }
    /**
     * Returns a URL relative to the given Location object.
     *
     * @param url URL.
     */
    async loadScript(url) {
        const browserSettings = __classPrivateFieldGet(this, _HTMLScriptElementScriptLoader_browserFrame, "f").page.context.browser.settings;
        const element = __classPrivateFieldGet(this, _HTMLScriptElementScriptLoader_element, "f");
        const async = element.getAttribute('async') !== null;
        if (!url || !element[PropertySymbol.isConnected]) {
            return;
        }
        let absoluteURL;
        try {
            absoluteURL = new URL(url, element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].location.href).href;
        }
        catch (error) {
            return;
        }
        if (__classPrivateFieldGet(this, _HTMLScriptElementScriptLoader_loadedScriptURL, "f") === absoluteURL) {
            return;
        }
        if (browserSettings.disableJavaScriptFileLoading ||
            browserSettings.disableJavaScriptEvaluation) {
            if (browserSettings.handleDisabledFileLoadingAsSuccess) {
                element.dispatchEvent(new Event_js_1.default('load'));
            }
            else {
                WindowErrorUtility_js_1.default.dispatchError(element, new DOMException_js_1.default(`Failed to load external script "${absoluteURL}". JavaScript file loading is disabled.`, DOMExceptionNameEnum_js_1.default.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch_js_1.default({
            browserFrame: __classPrivateFieldGet(this, _HTMLScriptElementScriptLoader_browserFrame, "f"),
            window: element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow]
        });
        let code = null;
        let error = null;
        __classPrivateFieldSet(this, _HTMLScriptElementScriptLoader_loadedScriptURL, absoluteURL, "f");
        if (async) {
            const readyStateManager = element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow][PropertySymbol.readyStateManager];
            readyStateManager.startTask();
            try {
                code = await resourceFetch.fetch(absoluteURL);
            }
            catch (e) {
                error = e;
            }
            readyStateManager.endTask();
        }
        else {
            try {
                code = resourceFetch.fetchSync(absoluteURL);
            }
            catch (e) {
                error = e;
            }
        }
        if (error) {
            WindowErrorUtility_js_1.default.dispatchError(element, error);
        }
        else {
            element[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = element;
            code = '//# sourceURL=' + absoluteURL + '\n' + code;
            if (browserSettings.disableErrorCapturing ||
                browserSettings.errorCapture !== BrowserErrorCaptureEnum_js_1.default.tryAndCatch) {
                element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code);
            }
            else {
                WindowErrorUtility_js_1.default.captureError(element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow], () => element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code));
            }
            element[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = null;
            element.dispatchEvent(new Event_js_1.default('load'));
        }
    }
}
_HTMLScriptElementScriptLoader_element = new WeakMap(), _HTMLScriptElementScriptLoader_browserFrame = new WeakMap(), _HTMLScriptElementScriptLoader_loadedScriptURL = new WeakMap();
exports.default = HTMLScriptElementScriptLoader;
//# sourceMappingURL=HTMLScriptElementScriptLoader.cjs.map