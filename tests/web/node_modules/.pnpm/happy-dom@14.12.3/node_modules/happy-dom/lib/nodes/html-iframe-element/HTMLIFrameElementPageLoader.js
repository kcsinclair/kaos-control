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
var _HTMLIFrameElementPageLoader_element, _HTMLIFrameElementPageLoader_contentWindowContainer, _HTMLIFrameElementPageLoader_browserParentFrame, _HTMLIFrameElementPageLoader_browserIFrame, _HTMLIFrameElementPageLoader_srcdoc;
import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import BrowserFrameURL from '../../browser/utilities/BrowserFrameURL.js';
import BrowserFrameFactory from '../../browser/utilities/BrowserFrameFactory.js';
/**
 * HTML Iframe page loader.
 */
class HTMLIFrameElementPageLoader {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Iframe element.
     * @param options.browserParentFrame Main browser frame.
     * @param options.contentWindowContainer Content window container.
     * @param options.contentWindowContainer.window Content window.
     */
    constructor(options) {
        _HTMLIFrameElementPageLoader_element.set(this, void 0);
        _HTMLIFrameElementPageLoader_contentWindowContainer.set(this, void 0);
        _HTMLIFrameElementPageLoader_browserParentFrame.set(this, void 0);
        _HTMLIFrameElementPageLoader_browserIFrame.set(this, void 0);
        _HTMLIFrameElementPageLoader_srcdoc.set(this, null);
        __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_element, options.element, "f");
        __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_contentWindowContainer, options.contentWindowContainer, "f");
        __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_browserParentFrame, options.browserParentFrame, "f");
    }
    /**
     * Loads an iframe page.
     */
    loadPage() {
        if (!__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f")[PropertySymbol.isConnected]) {
            this.unloadPage();
            return;
        }
        const srcdoc = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f").getAttribute('srcdoc');
        const window = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f")[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow];
        if (srcdoc !== null) {
            if (__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_srcdoc, "f") === srcdoc) {
                return;
            }
            this.unloadPage();
            __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_browserIFrame, BrowserFrameFactory.createChildFrame(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f")), "f");
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").url = 'about:srcdoc';
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_contentWindowContainer, "f").window = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window;
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.top = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f").window.top;
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.parent = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f").window;
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.document.open();
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.document.write(srcdoc);
            __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_srcdoc, srcdoc, "f");
            __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f")[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].requestAnimationFrame(() => __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f").dispatchEvent(new Event('load')));
            return;
        }
        if (__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_srcdoc, "f") !== null) {
            this.unloadPage();
        }
        const originURL = __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f").window.location;
        const targetURL = BrowserFrameURL.getRelativeURL(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f"), __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f").src);
        if (__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f") && __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.location.href === targetURL.href) {
            return;
        }
        if (__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f").page.context.browser.settings.disableIframePageLoading) {
            WindowErrorUtility.dispatchError(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f"), new DOMException(`Failed to load iframe page "${targetURL.href}". Iframe page loading is disabled.`, DOMExceptionNameEnum.notSupportedError));
            return;
        }
        // Iframes has a special rule for CORS and doesn't allow access between frames when the origin is different.
        const isSameOrigin = originURL.origin === targetURL.origin || targetURL.origin === 'null';
        const parentWindow = isSameOrigin ? window : new CrossOriginBrowserWindow(window);
        __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_browserIFrame, __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f") ?? BrowserFrameFactory.createChildFrame(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserParentFrame, "f")), "f");
        __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.top =
            parentWindow;
        __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window.parent =
            parentWindow;
        __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f")
            .goto(targetURL.href, {
            referrer: originURL.origin,
            referrerPolicy: __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f").referrerPolicy
        })
            .then(() => __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f").dispatchEvent(new Event('load')))
            .catch((error) => WindowErrorUtility.dispatchError(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_element, "f"), error));
        __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_contentWindowContainer, "f").window = isSameOrigin
            ? __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window
            : new CrossOriginBrowserWindow(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f").window, window);
    }
    /**
     * Unloads an iframe page.
     */
    unloadPage() {
        if (__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f")) {
            BrowserFrameFactory.destroyFrame(__classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_browserIFrame, "f"));
            __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_browserIFrame, null, "f");
        }
        __classPrivateFieldGet(this, _HTMLIFrameElementPageLoader_contentWindowContainer, "f").window = null;
        __classPrivateFieldSet(this, _HTMLIFrameElementPageLoader_srcdoc, null, "f");
    }
}
_HTMLIFrameElementPageLoader_element = new WeakMap(), _HTMLIFrameElementPageLoader_contentWindowContainer = new WeakMap(), _HTMLIFrameElementPageLoader_browserParentFrame = new WeakMap(), _HTMLIFrameElementPageLoader_browserIFrame = new WeakMap(), _HTMLIFrameElementPageLoader_srcdoc = new WeakMap();
export default HTMLIFrameElementPageLoader;
//# sourceMappingURL=HTMLIFrameElementPageLoader.js.map