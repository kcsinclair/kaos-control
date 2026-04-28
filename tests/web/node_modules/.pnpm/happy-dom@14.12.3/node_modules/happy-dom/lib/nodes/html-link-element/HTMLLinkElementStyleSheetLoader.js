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
var _HTMLLinkElementStyleSheetLoader_element, _HTMLLinkElementStyleSheetLoader_browserFrame, _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL;
import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ResourceFetch from '../../fetch/ResourceFetch.js';
import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
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
                element.dispatchEvent(new Event('load'));
            }
            else {
                WindowErrorUtility.dispatchError(element, new DOMException(`Failed to load external stylesheet "${absoluteURL}". CSS file loading is disabled.`, DOMExceptionNameEnum.notSupportedError));
            }
            return;
        }
        const resourceFetch = new ResourceFetch({
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
            WindowErrorUtility.dispatchError(element, error);
        }
        else {
            const styleSheet = new CSSStyleSheet();
            styleSheet.replaceSync(code);
            element[PropertySymbol.sheet] = styleSheet;
            element.dispatchEvent(new Event('load'));
        }
    }
}
_HTMLLinkElementStyleSheetLoader_element = new WeakMap(), _HTMLLinkElementStyleSheetLoader_browserFrame = new WeakMap(), _HTMLLinkElementStyleSheetLoader_loadedStyleSheetURL = new WeakMap();
export default HTMLLinkElementStyleSheetLoader;
//# sourceMappingURL=HTMLLinkElementStyleSheetLoader.js.map