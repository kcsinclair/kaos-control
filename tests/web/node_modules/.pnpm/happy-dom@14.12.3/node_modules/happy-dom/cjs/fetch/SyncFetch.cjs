"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var _SyncFetch_browserFrame, _SyncFetch_window;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const child_process_1 = __importDefault(require("child_process"));
const Headers_js_1 = __importDefault(require("./Headers.cjs"));
const CachedResponseStateEnum_js_1 = __importDefault(require("./cache/response/CachedResponseStateEnum.cjs"));
const FetchRequestReferrerUtility_js_1 = __importDefault(require("./utilities/FetchRequestReferrerUtility.cjs"));
const FetchRequestValidationUtility_js_1 = __importDefault(require("./utilities/FetchRequestValidationUtility.cjs"));
const DataURIParser_js_1 = __importDefault(require("./data-uri/DataURIParser.cjs"));
const SyncFetchScriptBuilder_js_1 = __importDefault(require("./utilities/SyncFetchScriptBuilder.cjs"));
const FetchRequestHeaderUtility_js_1 = __importDefault(require("./utilities/FetchRequestHeaderUtility.cjs"));
const FetchResponseHeaderUtility_js_1 = __importDefault(require("./utilities/FetchResponseHeaderUtility.cjs"));
const zlib_1 = __importDefault(require("zlib"));
const FetchResponseRedirectUtility_js_1 = __importDefault(require("./utilities/FetchResponseRedirectUtility.cjs"));
const FetchCORSUtility_js_1 = __importDefault(require("./utilities/FetchCORSUtility.cjs"));
const Fetch_js_1 = __importDefault(require("./Fetch.cjs"));
/**
 * Handles synchrounous fetch requests.
 */
class SyncFetch {
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     * @param options.url URL.
     * @param [options.init] Init.
     * @param [options.redirectCount] Redirect count.
     * @param [options.contentType] Content Type.
     * @param [options.disableCache] Disables the use of cached responses. It will still store the response in the cache.
     * @param [options.disableCrossOriginPolicy] Disables the Cross-Origin policy.
     */
    constructor(options) {
        this.redirectCount = 0;
        _SyncFetch_browserFrame.set(this, void 0);
        _SyncFetch_window.set(this, void 0);
        __classPrivateFieldSet(this, _SyncFetch_browserFrame, options.browserFrame, "f");
        __classPrivateFieldSet(this, _SyncFetch_window, options.window, "f");
        this.request =
            typeof options.url === 'string' || options.url instanceof URL_js_1.default
                ? new options.browserFrame.window.Request(options.url, options.init)
                : options.url;
        if (options.contentType) {
            this.request[PropertySymbol.contentType] = options.contentType;
        }
        this.redirectCount = options.redirectCount ?? 0;
        this.disableCache = options.disableCache ?? false;
        this.disableCrossOriginPolicy = options.disableCrossOriginPolicy ?? false;
    }
    /**
     * Sends request.
     *
     * @returns Response.
     */
    send() {
        FetchRequestReferrerUtility_js_1.default.prepareRequest(new URL_js_1.default(__classPrivateFieldGet(this, _SyncFetch_window, "f").location.href), this.request);
        FetchRequestValidationUtility_js_1.default.validateSchema(this.request);
        if (this.request.signal.aborted) {
            throw new DOMException_js_1.default('The operation was aborted.', DOMExceptionNameEnum_js_1.default.abortError);
        }
        if (this.request[PropertySymbol.url].protocol === 'data:') {
            const result = DataURIParser_js_1.default.parse(this.request.url);
            return {
                status: 200,
                statusText: 'OK',
                ok: true,
                url: this.request.url,
                redirected: false,
                headers: new Headers_js_1.default({ 'Content-Type': result.type }),
                body: result.buffer
            };
        }
        // Security check for "https" to "http" requests.
        if (this.request[PropertySymbol.url].protocol === 'http:' &&
            __classPrivateFieldGet(this, _SyncFetch_window, "f").location.protocol === 'https:') {
            throw new DOMException_js_1.default(`Mixed Content: The page at '${__classPrivateFieldGet(this, _SyncFetch_window, "f").location.href}' was loaded over HTTPS, but requested an insecure XMLHttpRequest endpoint '${this.request.url}'. This request has been blocked; the content must be served over HTTPS.`, DOMExceptionNameEnum_js_1.default.securityError);
        }
        const cachedResponse = this.getCachedResponse();
        if (cachedResponse) {
            return cachedResponse;
        }
        if (!this.compliesWithCrossOriginPolicy()) {
            __classPrivateFieldGet(this, _SyncFetch_window, "f").console.warn(`Cross-Origin Request Blocked: The Same Origin Policy dissallows reading the remote resource at "${this.request.url}".`);
            throw new DOMException_js_1.default(`Cross-Origin Request Blocked: The Same Origin Policy dissallows reading the remote resource at "${this.request.url}".`, DOMExceptionNameEnum_js_1.default.networkError);
        }
        return this.sendRequest();
    }
    /**
     * Returns cached response.
     *
     * @returns Response.
     */
    getCachedResponse() {
        if (this.disableCache) {
            return null;
        }
        let cachedResponse = __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f").page.context.responseCache.get(this.request);
        if (!cachedResponse || cachedResponse.response.waitingForBody) {
            return null;
        }
        if (cachedResponse.state === CachedResponseStateEnum_js_1.default.stale) {
            const headers = new Headers_js_1.default(cachedResponse.request.headers);
            if (cachedResponse.etag) {
                headers.set('If-None-Match', cachedResponse.etag);
            }
            else {
                if (!cachedResponse.lastModified) {
                    return null;
                }
                headers.set('If-Modified-Since', new Date(cachedResponse.lastModified).toUTCString());
            }
            if (cachedResponse.etag || !cachedResponse.staleWhileRevalidate) {
                const fetch = new SyncFetch({
                    browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
                    window: __classPrivateFieldGet(this, _SyncFetch_window, "f"),
                    url: this.request.url,
                    init: { headers, method: cachedResponse.request.method },
                    disableCache: true,
                    disableCrossOriginPolicy: true
                });
                const validateResponse = fetch.send();
                const body = validateResponse.status !== 304 ? validateResponse.body : null;
                cachedResponse = __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f").page.context.responseCache.add(this.request, {
                    ...validateResponse,
                    body,
                    waitingForBody: false
                });
                if (validateResponse.status !== 304) {
                    return validateResponse;
                }
            }
            else {
                const fetch = new Fetch_js_1.default({
                    browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
                    window: __classPrivateFieldGet(this, _SyncFetch_window, "f"),
                    url: this.request.url,
                    init: { headers, method: cachedResponse.request.method },
                    disableCache: true,
                    disableCrossOriginPolicy: true
                });
                fetch.send().then((response) => {
                    response.buffer().then((body) => {
                        __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f").page.context.responseCache.add(this.request, {
                            ...response,
                            body,
                            waitingForBody: false
                        });
                    });
                });
            }
        }
        if (!cachedResponse || cachedResponse.response.waitingForBody) {
            return null;
        }
        return {
            status: cachedResponse.response.status,
            statusText: cachedResponse.response.statusText,
            ok: true,
            url: cachedResponse.response.url,
            // TODO: Do we need to add support for redirected responses to the cache?
            redirected: false,
            headers: cachedResponse.response.headers,
            body: cachedResponse.response.body
        };
    }
    /**
     * Checks if the request complies with the Cross-Origin policy.
     *
     * @returns True if it complies with the policy.
     */
    compliesWithCrossOriginPolicy() {
        if (this.disableCrossOriginPolicy ||
            !FetchCORSUtility_js_1.default.isCORS(__classPrivateFieldGet(this, _SyncFetch_window, "f").location.href, this.request[PropertySymbol.url])) {
            return true;
        }
        const cachedPreflightResponse = __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f").page.context.preflightResponseCache.get(this.request);
        if (cachedPreflightResponse) {
            if (cachedPreflightResponse.allowOrigin !== '*' &&
                cachedPreflightResponse.allowOrigin !== __classPrivateFieldGet(this, _SyncFetch_window, "f").location.origin) {
                return false;
            }
            if (cachedPreflightResponse.allowMethods.length !== 0 &&
                !cachedPreflightResponse.allowMethods.includes(this.request.method)) {
                return false;
            }
            return true;
        }
        const requestHeaders = [];
        for (const [header] of this.request.headers) {
            requestHeaders.push(header);
        }
        const fetch = new SyncFetch({
            browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
            window: __classPrivateFieldGet(this, _SyncFetch_window, "f"),
            url: this.request.url,
            init: {
                method: 'OPTIONS',
                headers: new Headers_js_1.default({
                    'Access-Control-Request-Method': this.request.method,
                    'Access-Control-Request-Headers': requestHeaders.join(', ')
                })
            },
            disableCache: true,
            disableCrossOriginPolicy: true
        });
        const response = fetch.send();
        if (!response.ok) {
            return false;
        }
        const allowOrigin = response.headers.get('Access-Control-Allow-Origin');
        if (!allowOrigin) {
            return false;
        }
        if (allowOrigin !== '*' && allowOrigin !== __classPrivateFieldGet(this, _SyncFetch_window, "f").location.origin) {
            return false;
        }
        const allowMethods = [];
        if (response.headers.has('Access-Control-Allow-Methods')) {
            const allowMethodsHeader = response.headers.get('Access-Control-Allow-Methods');
            if (allowMethodsHeader !== '*') {
                for (const method of allowMethodsHeader.split(',')) {
                    allowMethods.push(method.trim().toUpperCase());
                }
            }
        }
        if (allowMethods.length !== 0 && !allowMethods.includes(this.request.method)) {
            return false;
        }
        // TODO: Add support for more Access-Control-Allow-* headers.
        return true;
    }
    /**
     * Sends request.
     *
     * @returns Response.
     */
    sendRequest() {
        if (!this.request[PropertySymbol.bodyBuffer] && this.request.body) {
            throw new DOMException_js_1.default(`Streams are not supported as request body for synchrounous requests.`, DOMExceptionNameEnum_js_1.default.notSupportedError);
        }
        const script = SyncFetchScriptBuilder_js_1.default.getScript({
            url: this.request[PropertySymbol.url],
            method: this.request.method,
            headers: FetchRequestHeaderUtility_js_1.default.getRequestHeaders({
                browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
                window: __classPrivateFieldGet(this, _SyncFetch_window, "f"),
                request: this.request
            }),
            body: this.request[PropertySymbol.bodyBuffer]
        });
        // Start the other Node Process, executing this string
        const content = child_process_1.default.execFileSync(process.argv[0], ['-e', script], {
            encoding: 'buffer',
            maxBuffer: 1024 * 1024 * 1024 // TODO: Consistent buffer size: 1GB.
        });
        // If content length is 0, then there was an error
        if (!content.length) {
            throw new DOMException_js_1.default(`Synchronous fetch to "${this.request.url}" failed.`, DOMExceptionNameEnum_js_1.default.networkError);
        }
        const { error, incomingMessage } = JSON.parse(content.toString());
        if (error) {
            throw new DOMException_js_1.default(`Synchronous fetch to "${this.request.url}" failed. Error: ${error}`, DOMExceptionNameEnum_js_1.default.networkError);
        }
        const headers = FetchResponseHeaderUtility_js_1.default.parseResponseHeaders({
            browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
            requestURL: this.request[PropertySymbol.url],
            rawHeaders: incomingMessage.rawHeaders
        });
        const response = {
            status: incomingMessage.statusCode,
            statusText: incomingMessage.statusMessage,
            ok: incomingMessage.statusCode >= 200 && incomingMessage.statusCode < 300,
            url: this.request.url,
            redirected: this.redirectCount > 0,
            headers,
            body: this.parseIResponseBody({
                headers,
                status: incomingMessage.statusCode,
                body: Buffer.from(incomingMessage.data, 'base64')
            })
        };
        const redirectedResponse = this.handleRedirectResponse(response) || response;
        if (!this.disableCache && !redirectedResponse.redirected) {
            __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f").page.context.responseCache.add(this.request, {
                status: redirectedResponse.status,
                statusText: redirectedResponse.statusText,
                url: redirectedResponse.url,
                headers: redirectedResponse.headers,
                body: redirectedResponse.body,
                waitingForBody: false
            });
        }
        return redirectedResponse;
    }
    /**
     * Parses response body.
     *
     * @param options Options.
     * @param options.headers Headers.
     * @param options.status Status.
     * @param options.body Body.
     * @returns Parsed body.
     */
    parseIResponseBody(options) {
        const contentEncodingHeader = options.headers.get('Content-Encoding');
        if (this.request.method === 'HEAD' ||
            contentEncodingHeader === null ||
            options.status === 204 ||
            options.status === 304) {
            return options.body;
        }
        try {
            // For GZip
            if (contentEncodingHeader === 'gzip' || contentEncodingHeader === 'x-gzip') {
                // Be less strict when decoding compressed responses by using Z_SYNC_FLUSH.
                // Sometimes servers send slightly invalid responses that are still accepted by common browsers.
                // "cURL" always uses Z_SYNC_FLUSH.
                return zlib_1.default.gunzipSync(options.body, {
                    flush: zlib_1.default.constants.Z_SYNC_FLUSH,
                    finishFlush: zlib_1.default.constants.Z_SYNC_FLUSH
                });
            }
            // For Deflate
            if (contentEncodingHeader === 'deflate' || contentEncodingHeader === 'x-deflate') {
                return zlib_1.default.inflateSync(options.body);
            }
            // For BR
            if (contentEncodingHeader === 'br') {
                return zlib_1.default.brotliDecompressSync(options.body);
            }
        }
        catch (error) {
            throw new DOMException_js_1.default(`Failed to read response body. Error: ${error.message}.`, DOMExceptionNameEnum_js_1.default.encodingError);
        }
        return options.body;
    }
    /**
     * Handles redirect response.
     *
     * @param response Response.
     * @returns Redirected response or null.
     */
    handleRedirectResponse(response) {
        if (!FetchResponseRedirectUtility_js_1.default.isRedirect(response.status)) {
            return null;
        }
        switch (this.request.redirect) {
            case 'error':
                throw new DOMException_js_1.default(`URI requested responds with a redirect, redirect mode is set to "error": ${this.request.url}`, DOMExceptionNameEnum_js_1.default.abortError);
            case 'manual':
                return null;
            case 'follow':
                const locationHeader = response.headers.get('Location');
                const shouldBecomeGetRequest = response.status === 303 ||
                    ((response.status === 301 || response.status === 302) && this.request.method === 'POST');
                let locationURL = null;
                if (locationHeader !== null) {
                    try {
                        locationURL = new URL_js_1.default(locationHeader, this.request.url);
                    }
                    catch {
                        throw new DOMException_js_1.default(`URI requested responds with an invalid redirect URL: ${locationHeader}`, DOMExceptionNameEnum_js_1.default.uriMismatchError);
                    }
                }
                if (locationURL === null) {
                    return null;
                }
                if (FetchResponseRedirectUtility_js_1.default.isMaxRedirectsReached(this.redirectCount)) {
                    throw new DOMException_js_1.default(`Maximum redirects reached at: ${this.request.url}`, DOMExceptionNameEnum_js_1.default.networkError);
                }
                const headers = new Headers_js_1.default(this.request.headers);
                const requestInit = {
                    method: this.request.method,
                    signal: this.request.signal,
                    referrer: this.request.referrer,
                    referrerPolicy: this.request.referrerPolicy,
                    credentials: this.request.credentials,
                    headers,
                    body: this.request[PropertySymbol.bodyBuffer]
                };
                if (this.request.credentials === 'omit' ||
                    (this.request.credentials === 'same-origin' &&
                        FetchCORSUtility_js_1.default.isCORS(__classPrivateFieldGet(this, _SyncFetch_window, "f").location.href, locationURL))) {
                    headers.delete('authorization');
                    headers.delete('www-authenticate');
                    headers.delete('cookie');
                    headers.delete('cookie2');
                }
                if (shouldBecomeGetRequest) {
                    requestInit.method = 'GET';
                    requestInit.body = undefined;
                    headers.delete('Content-Length');
                    headers.delete('Content-Type');
                }
                const responseReferrerPolicy = FetchRequestReferrerUtility_js_1.default.getReferrerPolicyFromHeader(headers);
                if (responseReferrerPolicy) {
                    requestInit.referrerPolicy = responseReferrerPolicy;
                }
                const fetch = new SyncFetch({
                    browserFrame: __classPrivateFieldGet(this, _SyncFetch_browserFrame, "f"),
                    window: __classPrivateFieldGet(this, _SyncFetch_window, "f"),
                    url: locationURL,
                    init: requestInit,
                    redirectCount: this.redirectCount + 1,
                    contentType: !shouldBecomeGetRequest
                        ? this.request[PropertySymbol.contentType]
                        : undefined
                });
                return fetch.send();
            default:
                throw new DOMException_js_1.default(`Redirect option '${this.request.redirect}' is not a valid value of IRequestRedirect`);
        }
    }
}
_SyncFetch_browserFrame = new WeakMap(), _SyncFetch_window = new WeakMap();
exports.default = SyncFetch;
//# sourceMappingURL=SyncFetch.cjs.map