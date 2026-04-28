"use strict";
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
var _DetachedWindowAPI_browserFrame;
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * API for detached windows to be able to access features of the browser.
 */
class DetachedWindowAPI {
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     */
    constructor(browserFrame) {
        _DetachedWindowAPI_browserFrame.set(this, void 0);
        __classPrivateFieldSet(this, _DetachedWindowAPI_browserFrame, browserFrame, "f");
    }
    /**
     * Returns settings.
     *
     * @returns Settings.
     */
    get settings() {
        return __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").page.context.browser.settings;
    }
    /**
     * Returns virtual console printer.
     *
     * @returns Virtual console printer.
     */
    get virtualConsolePrinter() {
        return __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").page.virtualConsolePrinter;
    }
    /**
     * Waits for all async tasks to complete.
     *
     * @returns Promise.
     */
    waitUntilComplete() {
        return __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").waitUntilComplete();
    }
    /**
     * Waits for all async tasks to complete.
     *
     * @deprecated Use waitUntilComplete() instead.
     * @returns Promise.
     */
    whenAsyncComplete() {
        return this.waitUntilComplete();
    }
    /**
     * Aborts all async tasks.
     */
    abort() {
        return __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").abort();
    }
    /**
     * Aborts all async tasks.
     *
     * @deprecated Use abort() instead.
     */
    cancelAsync() {
        return this.abort();
    }
    /**
     * Aborts all async tasks and closes the window.
     */
    close() {
        return __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").page.close();
    }
    /**
     * Sets the URL without navigating the browser.
     *
     * @param url URL.
     */
    setURL(url) {
        __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").url = url;
    }
    /**
     * Sets the viewport.
     *
     * @param viewport Viewport.
     */
    setViewport(viewport) {
        __classPrivateFieldGet(this, _DetachedWindowAPI_browserFrame, "f").page.setViewport(viewport);
    }
    /**
     * Sets the window size.
     *
     * @deprecated Use setViewport() instead.
     * @param options Options.
     * @param options.width Width.
     * @param options.height Height.
     */
    setWindowSize(options) {
        this.setViewport({
            width: options?.width,
            height: options?.height
        });
    }
    /**
     * Sets the window width.
     *
     * @deprecated Use setViewport() instead.
     * @param width Width.
     */
    setInnerWidth(width) {
        this.setViewport({ width });
    }
    /**
     * Sets the window height.
     *
     * @deprecated Use setViewport() instead.
     * @param height Height.
     */
    setInnerHeight(height) {
        this.setViewport({ height });
    }
}
_DetachedWindowAPI_browserFrame = new WeakMap();
exports.default = DetachedWindowAPI;
//# sourceMappingURL=DetachedWindowAPI.cjs.map