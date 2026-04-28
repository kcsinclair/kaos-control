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
var _HTMLScriptElementScriptLoader_element, _HTMLScriptElementScriptLoader_browserFrame, _HTMLScriptElementScriptLoader_loadedScriptURL;
import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import ResourceFetch from '../../fetch/ResourceFetch.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import BrowserErrorCaptureEnum from '../../browser/enums/BrowserErrorCaptureEnum.js';
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
                element.dispatchEvent(new Event('load'));
            }
            else {
                WindowErrorUtility.dispatchError(element, new DOMException(`Failed to load external script "${absoluteURL}". JavaScript file loading is disabled.`, DOMExceptionNameEnum.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch({
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
            WindowErrorUtility.dispatchError(element, error);
        }
        else {
            element[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = element;
            code = '//# sourceURL=' + absoluteURL + '\n' + code;
            if (browserSettings.disableErrorCapturing ||
                browserSettings.errorCapture !== BrowserErrorCaptureEnum.tryAndCatch) {
                element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code);
            }
            else {
                WindowErrorUtility.captureError(element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow], () => element[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code));
            }
            element[PropertySymbol.ownerDocument][PropertySymbol.currentScript] = null;
            element.dispatchEvent(new Event('load'));
        }
    }
}
_HTMLScriptElementScriptLoader_element = new WeakMap(), _HTMLScriptElementScriptLoader_browserFrame = new WeakMap(), _HTMLScriptElementScriptLoader_loadedScriptURL = new WeakMap();
export default HTMLScriptElementScriptLoader;
//# sourceMappingURL=HTMLScriptElementScriptLoader.js.map