"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DetachedBrowserContext_js_1 = __importDefault(require("./DetachedBrowserContext.cjs"));
const BrowserSettingsFactory_js_1 = __importDefault(require("../BrowserSettingsFactory.cjs"));
/**
 * Detached browser used when constructing a Window instance without a browser.
 *
 * Much of the interface for the browser has been taken from Puppeteer and Playwright, so that the API is familiar.
 */
class DetachedBrowser {
    /**
     * Constructor.
     *
     * @param windowClass Window class.
     * @param [options] Options.
     * @param [options.settings] Browser settings.
     * @param [options.console] Console.
     */
    constructor(windowClass, options) {
        this.windowClass = windowClass;
        this.console = options?.console || null;
        this.settings = BrowserSettingsFactory_js_1.default.createSettings(options?.settings);
        this.contexts = [];
        this.contexts.push(new DetachedBrowserContext_js_1.default(this));
    }
    /**
     * Returns the default context.
     *
     * @returns Default context.
     */
    get defaultContext() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        return this.contexts[0];
    }
    /**
     * Aborts all ongoing operations and destroys the browser.
     */
    async close() {
        await Promise.all(this.contexts.slice().map((context) => context.close()));
        this.contexts = [];
        this.console = null;
        this.windowClass = null;
    }
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     *
     * @returns Promise.
     */
    async waitUntilComplete() {
        await Promise.all(this.contexts.map((page) => page.waitUntilComplete()));
    }
    /**
     * Aborts all ongoing operations.
     */
    abort() {
        // Using Promise instead of async/await to prevent microtask
        return new Promise((resolve, reject) => {
            if (!this.contexts.length) {
                resolve();
                return;
            }
            Promise.all(this.contexts.slice().map((context) => context.abort()))
                .then(() => resolve())
                .catch((error) => reject(error));
        });
    }
    /**
     * Creates a new incognito context.
     */
    newIncognitoContext() {
        throw new Error('Not possible to create a new context on a detached browser.');
    }
    /**
     * Creates a new page.
     *
     * @returns Page.
     */
    newPage() {
        if (this.contexts.length === 0) {
            throw new Error('No default context. The browser has been closed.');
        }
        return this.contexts[0].newPage();
    }
}
exports.default = DetachedBrowser;
//# sourceMappingURL=DetachedBrowser.cjs.map