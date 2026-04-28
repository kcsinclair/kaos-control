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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _ResourceFetch_browserFrame;
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const Fetch_js_1 = __importDefault(require("./Fetch.cjs"));
const SyncFetch_js_1 = __importDefault(require("./SyncFetch.cjs"));
/**
 * Helper class for performing fetch of resources.
 */
class ResourceFetch {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     */
    constructor(options) {
        _ResourceFetch_browserFrame.set(this, void 0);
        __classPrivateFieldSet(this, _ResourceFetch_browserFrame, options.browserFrame, "f");
        this.window = options.window;
    }
    /**
     * Returns resource data asynchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    async fetch(url) {
        const fetch = new Fetch_js_1.default({
            browserFrame: __classPrivateFieldGet(this, _ResourceFetch_browserFrame, "f"),
            window: this.window,
            url,
            disableCrossOriginPolicy: true
        });
        const response = await fetch.send();
        if (!response.ok) {
            throw new DOMException_js_1.default(`Failed to perform request to "${new URL_js_1.default(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
        }
        return await response.text();
    }
    /**
     * Returns resource data synchronously.
     *
     * @param url URL.
     * @returns Response.
     */
    fetchSync(url) {
        const fetch = new SyncFetch_js_1.default({
            browserFrame: __classPrivateFieldGet(this, _ResourceFetch_browserFrame, "f"),
            window: this.window,
            url,
            disableCrossOriginPolicy: true
        });
        const response = fetch.send();
        if (!response.ok) {
            throw new DOMException_js_1.default(`Failed to perform request to "${new URL_js_1.default(url, this.window.location.href).href}". Status ${response.status} ${response.statusText}.`);
        }
        return response.body.toString();
    }
}
_ResourceFetch_browserFrame = new WeakMap();
exports.default = ResourceFetch;
//# sourceMappingURL=ResourceFetch.cjs.map