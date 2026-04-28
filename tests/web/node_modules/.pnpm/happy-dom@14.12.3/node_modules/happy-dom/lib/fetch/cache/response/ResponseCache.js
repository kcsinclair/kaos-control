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
var _ResponseCache_entries;
import CachedResponseStateEnum from './CachedResponseStateEnum.js';
import Headers from '../../Headers.js';
const UPDATE_RESPONSE_HEADERS = ['Cache-Control', 'Last-Modified', 'Vary', 'ETag'];
/**
 * Fetch response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching
 * @see https://www.mnot.net/cache_docs/
 */
class ResponseCache {
    constructor() {
        _ResponseCache_entries.set(this, {});
    }
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request) {
        if (request.headers.get('Cache-Control')?.includes('no-cache')) {
            return null;
        }
        const url = request.url;
        if (__classPrivateFieldGet(this, _ResponseCache_entries, "f")[url]) {
            for (let i = 0, max = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].length; i < max; i++) {
                const entry = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url][i];
                let isMatch = entry.request.method === request.method;
                if (isMatch) {
                    for (const header of Object.keys(entry.vary)) {
                        const requestHeader = request.headers.get(header);
                        if (requestHeader !== null && entry.vary[header] !== requestHeader) {
                            isMatch = false;
                            break;
                        }
                    }
                }
                if (isMatch) {
                    if (entry.expires && entry.expires < Date.now()) {
                        if (entry.lastModified) {
                            entry.state = CachedResponseStateEnum.stale;
                        }
                        else if (!entry.etag) {
                            __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].splice(i, 1);
                            return null;
                        }
                    }
                    return entry;
                }
            }
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
        // We should only cache GET and HEAD requests.
        if ((request.method !== 'GET' && request.method !== 'HEAD') ||
            request.headers.get('Cache-Control')?.includes('no-cache')) {
            return null;
        }
        const url = request.url;
        let cachedResponse = this.get(request);
        if (response.status === 304) {
            if (!cachedResponse) {
                throw new Error('ResponseCache: Cached response not found.');
            }
            for (const name of UPDATE_RESPONSE_HEADERS) {
                if (response.headers.has(name)) {
                    cachedResponse.response.headers.set(name, response.headers.get(name));
                }
            }
            cachedResponse.cacheUpdateTime = Date.now();
            cachedResponse.state = CachedResponseStateEnum.fresh;
        }
        else {
            if (cachedResponse) {
                const index = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].indexOf(cachedResponse);
                if (index !== -1) {
                    __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].splice(index, 1);
                }
            }
            cachedResponse = {
                response: {
                    status: response.status,
                    statusText: response.statusText,
                    url: response.url,
                    headers: new Headers(response.headers),
                    // We need to wait for the body to be consumed and then populated if set to true (e.g. by using Response.text()).
                    waitingForBody: response.waitingForBody,
                    body: response.body ?? null
                },
                request: {
                    headers: request.headers,
                    method: request.method
                },
                vary: {},
                expires: null,
                etag: null,
                cacheUpdateTime: Date.now(),
                lastModified: null,
                mustRevalidate: false,
                staleWhileRevalidate: false,
                state: CachedResponseStateEnum.fresh
            };
            __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url] = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url] || [];
            __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].push(cachedResponse);
        }
        if (response.headers.has('Cache-Control')) {
            const age = response.headers.get('Age');
            for (const part of response.headers.get('Cache-Control').split(',')) {
                const [key, value] = part.trim().split('=');
                switch (key) {
                    case 'max-age':
                        cachedResponse.expires =
                            Date.now() + parseInt(value) * 1000 - (age ? parseInt(age) * 1000 : 0);
                        break;
                    case 'no-cache':
                    case 'no-store':
                        const index = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].indexOf(cachedResponse);
                        if (index !== -1) {
                            __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].splice(index, 1);
                        }
                        return null;
                    case 'must-revalidate':
                        cachedResponse.mustRevalidate = true;
                        break;
                    case 'stale-while-revalidate':
                        cachedResponse.staleWhileRevalidate = true;
                        break;
                }
            }
        }
        if (response.headers.has('Last-Modified')) {
            cachedResponse.lastModified = Date.parse(response.headers.get('Last-Modified'));
        }
        if (response.headers.has('Vary')) {
            for (const header of response.headers.get('Vary').split(',')) {
                const name = header.trim();
                const value = request.headers.get(name);
                if (value) {
                    cachedResponse.vary[name] = value;
                }
            }
        }
        if (response.headers.has('ETag')) {
            cachedResponse.etag = response.headers.get('ETag');
        }
        if (!cachedResponse.expires) {
            const expires = response.headers.get('Expires');
            if (expires) {
                cachedResponse.expires = Date.parse(expires);
            }
        }
        // Cache is invalid if it has expired and doesn't have an ETag.
        if (!cachedResponse.etag && (!cachedResponse.expires || cachedResponse.expires < Date.now())) {
            const index = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].indexOf(cachedResponse);
            if (index !== -1) {
                __classPrivateFieldGet(this, _ResponseCache_entries, "f")[url].splice(index, 1);
            }
            return null;
        }
        return cachedResponse;
    }
    /**
     * Clears the cache.
     *
     * @param [options] Options.
     * @param [options.url] URL.
     * @param [options.toTime] Removes all entries that are older than this time. Time in MS.
     */
    clear(options) {
        if (options) {
            if (options.toTime) {
                for (const key of options.url ? [options.url] : Object.keys(__classPrivateFieldGet(this, _ResponseCache_entries, "f"))) {
                    if (__classPrivateFieldGet(this, _ResponseCache_entries, "f")[key]) {
                        for (let i = 0, max = __classPrivateFieldGet(this, _ResponseCache_entries, "f")[key].length; i < max; i++) {
                            if (__classPrivateFieldGet(this, _ResponseCache_entries, "f")[key][i].cacheUpdateTime < options.toTime) {
                                __classPrivateFieldGet(this, _ResponseCache_entries, "f")[key].splice(i, 1);
                                i--;
                                max--;
                            }
                        }
                    }
                }
            }
            else if (options.url) {
                delete __classPrivateFieldGet(this, _ResponseCache_entries, "f")[options.url];
            }
        }
        else {
            __classPrivateFieldSet(this, _ResponseCache_entries, {}, "f");
        }
    }
}
_ResponseCache_entries = new WeakMap();
export default ResponseCache;
//# sourceMappingURL=ResponseCache.js.map