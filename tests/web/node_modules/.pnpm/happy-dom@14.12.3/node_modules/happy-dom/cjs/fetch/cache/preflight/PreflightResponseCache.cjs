"use strict";
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _PreflightResponseCache_entries;
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Fetch preflight response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
 */
class PreflightResponseCache {
    constructor() {
        _PreflightResponseCache_entries.set(this, {});
    }
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request) {
        const cachedResponse = __classPrivateFieldGet(this, _PreflightResponseCache_entries, "f")[request.url];
        if (cachedResponse) {
            if (cachedResponse.expires < Date.now()) {
                delete __classPrivateFieldGet(this, _PreflightResponseCache_entries, "f")[request.url];
                return null;
            }
            return cachedResponse;
        }
        return null;
    }
    /**
     * Adds a cache entity.
     *
     * @param request Request.
     * @param response Response.
     * @returns Cached response.
     */
    add(request, response) {
        delete __classPrivateFieldGet(this, _PreflightResponseCache_entries, "f")[request.url];
        if (request.headers.get('Cache-Control')?.includes('no-cache')) {
            return null;
        }
        if (response.status < 200 || response.status >= 300) {
            return null;
        }
        const maxAge = response.headers.get('Access-Control-Max-Age');
        const allowOrigin = response.headers.get('Access-Control-Allow-Origin');
        if (!maxAge || !allowOrigin) {
            return null;
        }
        const allowMethods = [];
        if (response.headers.has('Access-Control-Allow-Methods')) {
            const allowMethodsHeader = response.headers.get('Access-Control-Allow-Methods');
            if (allowMethodsHeader !== '*') {
                for (const method of response.headers.get('Access-Control-Allow-Methods').split(',')) {
                    allowMethods.push(method.trim().toUpperCase());
                }
            }
        }
        const cachedResponse = {
            allowOrigin,
            allowMethods,
            expires: Date.now() + parseInt(maxAge) * 1000
        };
        if (isNaN(cachedResponse.expires) || cachedResponse.expires < Date.now()) {
            return null;
        }
        __classPrivateFieldGet(this, _PreflightResponseCache_entries, "f")[request.url] = cachedResponse;
        return cachedResponse;
    }
    /**
     * Clears the cache.
     */
    clear() {
        __classPrivateFieldSet(this, _PreflightResponseCache_entries, {}, "f");
    }
}
_PreflightResponseCache_entries = new WeakMap();
exports.default = PreflightResponseCache;
//# sourceMappingURL=PreflightResponseCache.cjs.map