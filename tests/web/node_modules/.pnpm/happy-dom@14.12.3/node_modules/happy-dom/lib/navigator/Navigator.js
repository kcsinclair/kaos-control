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
var _Navigator_ownerWindow, _Navigator_clipboard, _Navigator_permissions;
import MimeTypeArray from './MimeTypeArray.js';
import PluginArray from './PluginArray.js';
import Permissions from '../permissions/Permissions.js';
import Clipboard from '../clipboard/Clipboard.js';
import WindowBrowserSettingsReader from '../window/WindowBrowserSettingsReader.js';
/**
 * Browser Navigator API.
 *
 * Mocked information is taken from FireFox.
 *
 * Reference:
 * https://html.spec.whatwg.org/multipage/system-state.html#dom-navigator.
 */
class Navigator {
    /**
     * Constructor.
     *
     * @param ownerWindow Owner window.
     */
    constructor(ownerWindow) {
        _Navigator_ownerWindow.set(this, void 0);
        _Navigator_clipboard.set(this, void 0);
        _Navigator_permissions.set(this, void 0);
        __classPrivateFieldSet(this, _Navigator_ownerWindow, ownerWindow, "f");
        __classPrivateFieldSet(this, _Navigator_clipboard, new Clipboard(ownerWindow), "f");
        __classPrivateFieldSet(this, _Navigator_permissions, new Permissions(), "f");
    }
    /**
     * False if setting a cookie will be ignored and true otherwise.
     */
    get cookieEnabled() {
        return true;
    }
    /**
     * TODO: Not implemented.
     */
    get credentials() {
        return null;
    }
    /**
     * TODO: Not implemented.
     */
    get geolocation() {
        return null;
    }
    /**
     * String representing the preferred language of the user, usually the language of the browser UI.
     */
    get language() {
        return 'en-US';
    }
    /**
     * Array of string representing the user's preferred languages.
     */
    get languages() {
        return ['en-US', 'en'];
    }
    /**
     * TODO: Not implemented.
     */
    get locks() {
        return null;
    }
    /**
     * Maximum number of simultaneous touch contact points are supported by the current device.
     */
    get maxTouchPoints() {
        return (WindowBrowserSettingsReader.getSettings(__classPrivateFieldGet(this, _Navigator_ownerWindow, "f"))?.navigator.maxTouchPoints || 0);
    }
    /**
     * Number of logical processors available to run threads on the user's computer.
     */
    get hardwareConcurrency() {
        return 8;
    }
    /**
     * Browser app code name.
     */
    get appCodeName() {
        return 'Mozilla';
    }
    /**
     * Browser app name.
     */
    get appName() {
        return 'Netscape';
    }
    /**
     * Browser app version.
     */
    get appVersion() {
        const userAgent = this.userAgent;
        const index = userAgent.indexOf('/');
        return index > -1 ? userAgent.substring(index + 1) : '';
    }
    /**
     * Browser platform.
     */
    get platform() {
        const userAgent = this.userAgent;
        const indexStart = userAgent.indexOf('(');
        const indexEnd = userAgent.indexOf(')');
        return indexStart > -1 && indexEnd > -1 ? userAgent.substring(indexStart + 1, indexEnd) : '';
    }
    /**
     * Browser product.
     */
    get product() {
        return 'Gecko';
    }
    /**
     * Browser product sub.
     */
    get productSub() {
        return '20100101';
    }
    /**
     * Browser vendor.
     */
    get vendor() {
        return '';
    }
    /**
     * Browser vendor sub.
     */
    get vendorSub() {
        return '';
    }
    /**
     * Browser user agent.
     *
     * "appCodeName/appVersion number (Platform; Security; OS-or-CPU; Localization; rv: revision-version-number) product/productSub Application-Name Application-Name-version".
     */
    get userAgent() {
        return WindowBrowserSettingsReader.getSettings(__classPrivateFieldGet(this, _Navigator_ownerWindow, "f"))?.navigator.userAgent || '';
    }
    /**
     * Boolean value indicating whether the browser is working online.
     */
    get onLine() {
        return true;
    }
    /**
     * Returns a Permissions object that can be used to query and update permission status of APIs covered by the Permissions API.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/permissions
     * @returns Permissions.
     */
    get permissions() {
        return __classPrivateFieldGet(this, _Navigator_permissions, "f");
    }
    /**
     * Returns a Clipboard object providing access to the contents of the system clipboard.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Navigator/clipboard
     * @returns Clipboard.
     */
    get clipboard() {
        return __classPrivateFieldGet(this, _Navigator_clipboard, "f");
    }
    /**
     * Boolean Indicates whether the user agent is controlled by automation.
     */
    get webdriver() {
        return true;
    }
    /**
     * The user's Do Not Track setting, which indicates whether the user is requesting web sites and advertisers to not track them.
     *
     * The value of the property reflects that of the DNT HTTP header, i.e. Values of "1", "0", or "unspecified".
     */
    get doNotTrack() {
        return 'unspecified';
    }
    /**
     * Browser mime-types.
     */
    get mimeTypes() {
        return new MimeTypeArray([]);
    }
    /**
     * Browser plugins.
     */
    get plugins() {
        return new PluginArray([]);
    }
    /**
     * Sends an HTTP POST request containing a small amount of data to a web server.
     *
     * @param url URL.
     * @param data Data.
     * @returns "true" if the user agent successfully queued the data for transfer. Otherwise, it returns "false".
     */
    sendBeacon(url, data) {
        __classPrivateFieldGet(this, _Navigator_ownerWindow, "f").fetch(url, {
            method: 'POST',
            body: data
        });
        return true;
    }
    /**
     * Returns the object as a string.
     *
     * @returns String.
     */
    toString() {
        return '[object Navigator]';
    }
}
_Navigator_ownerWindow = new WeakMap(), _Navigator_clipboard = new WeakMap(), _Navigator_permissions = new WeakMap();
export default Navigator;
//# sourceMappingURL=Navigator.js.map