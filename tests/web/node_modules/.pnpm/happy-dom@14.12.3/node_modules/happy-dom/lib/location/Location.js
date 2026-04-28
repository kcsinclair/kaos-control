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
var _Location_browserFrame, _Location_url, _a;
import HashChangeEvent from '../event/events/HashChangeEvent.js';
import * as PropertySymbol from '../PropertySymbol.js';
import { URL } from 'url';
/**
 * Location.
 */
export default class Location {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param url URL.
     */
    constructor(browserFrame, url) {
        // Public properties
        this[_a] = 'Location';
        // Private properties
        _Location_browserFrame.set(this, void 0);
        _Location_url.set(this, void 0);
        __classPrivateFieldSet(this, _Location_browserFrame, browserFrame, "f");
        __classPrivateFieldSet(this, _Location_url, new URL(url), "f");
    }
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash() {
        return __classPrivateFieldGet(this, _Location_url, "f").hash;
    }
    /**
     * Sets hash.
     *
     * @param hash Value.
     */
    set hash(hash) {
        const oldURL = __classPrivateFieldGet(this, _Location_url, "f").href;
        __classPrivateFieldGet(this, _Location_url, "f").hash = hash;
        const newURL = __classPrivateFieldGet(this, _Location_url, "f").href;
        if (newURL !== oldURL) {
            __classPrivateFieldGet(this, _Location_browserFrame, "f").window?.dispatchEvent(new HashChangeEvent('hashchange', { oldURL, newURL }));
        }
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        return __classPrivateFieldGet(this, _Location_url, "f").host;
    }
    /**
     * Sets host.
     *
     * @param host Value.
     */
    set host(host) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.host = host;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname() {
        return __classPrivateFieldGet(this, _Location_url, "f").hostname;
    }
    /**
     * Sets hostname.
     *
     * @param hostname Value.
     */
    set hostname(hostname) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.hostname = hostname;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Override set href.
     */
    get href() {
        return __classPrivateFieldGet(this, _Location_url, "f").href;
    }
    /**
     * Override set href.
     */
    set href(url) {
        __classPrivateFieldGet(this, _Location_browserFrame, "f").goto(url).catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Returns origin.
     *
     * @returns Origin.
     */
    get origin() {
        return __classPrivateFieldGet(this, _Location_url, "f").origin;
    }
    /**
     * Returns pathname
     *
     * @returns Pathname.
     */
    get pathname() {
        return __classPrivateFieldGet(this, _Location_url, "f").pathname;
    }
    /**
     * Sets pathname.
     *
     * @param pathname Value.
     */
    set pathname(pathname) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.pathname = pathname;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port() {
        return __classPrivateFieldGet(this, _Location_url, "f").port;
    }
    /**
     * Sets port.
     *
     * @param port Value.
     */
    set port(port) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.port = port;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol() {
        return __classPrivateFieldGet(this, _Location_url, "f").protocol;
    }
    /**
     * Sets protocol.
     *
     * @param protocol Value.
     */
    set protocol(protocol) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.protocol = protocol;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search() {
        return __classPrivateFieldGet(this, _Location_url, "f").search;
    }
    /**
     * Sets search.
     *
     * @param search Value.
     */
    set search(search) {
        const url = new URL(__classPrivateFieldGet(this, _Location_url, "f").href);
        url.search = search;
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(url.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Replaces the current resource with the one at the provided URL. The difference from the assign() method is that after using replace() the current page will not be saved in session History, meaning the user won't be able to use the back button to navigate to it.
     *
     * @param url URL.
     */
    replace(url) {
        this.href = url;
    }
    /**
     * Loads the resource at the URL provided in parameter.
     *
     * @param url URL.
     */
    assign(url) {
        this.href = url;
    }
    /**
     * Reloads the resource from the current URL.
     */
    reload() {
        __classPrivateFieldGet(this, _Location_browserFrame, "f")
            .goto(this.href)
            .catch((error) => __classPrivateFieldGet(this, _Location_browserFrame, "f").page.console.error(error));
    }
    /**
     * Replaces the current URL state with the provided one without navigating to the new URL.
     *
     * @param browserFrame Browser frame that must match the current one as validation.
     * @param url URL.
     */
    [(_Location_browserFrame = new WeakMap(), _Location_url = new WeakMap(), _a = Symbol.toStringTag, PropertySymbol.setURL)](browserFrame, url) {
        if (__classPrivateFieldGet(this, _Location_browserFrame, "f") !== browserFrame) {
            throw new Error('Failed to set URL. Browser frame mismatch.');
        }
        __classPrivateFieldGet(this, _Location_url, "f").href = url;
    }
    /**
     * Returns the URL as a string.
     *
     * @returns URL as a string.
     */
    toString() {
        return __classPrivateFieldGet(this, _Location_url, "f").toString();
    }
}
//# sourceMappingURL=Location.js.map