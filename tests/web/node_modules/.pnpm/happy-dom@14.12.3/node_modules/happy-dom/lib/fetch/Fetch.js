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
var _Fetch_browserFrame, _Fetch_window;
import * as PropertySymbol from '../PropertySymbol.js';
import Headers from './Headers.js';
import FetchRequestReferrerUtility from './utilities/FetchRequestReferrerUtility.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import HTTP from 'http';
import HTTPS from 'https';
import Zlib from 'zlib';
import URL from '../url/URL.js';
import Stream from 'stream';
import DataURIParser from './data-uri/DataURIParser.js';
import FetchCORSUtility from './utilities/FetchCORSUtility.js';
import Response from './Response.js';
import CachedResponseStateEnum from './cache/response/CachedResponseStateEnum.js';
import FetchRequestHeaderUtility from './utilities/FetchRequestHeaderUtility.js';
import FetchRequestValidationUtility from './utilities/FetchRequestValidationUtility.js';
import FetchResponseRedirectUtility from './utilities/FetchResponseRedirectUtility.js';
import FetchResponseHeaderUtility from './utilities/FetchResponseHeaderUtility.js';
import FetchHTTPSCertificate from './certificate/FetchHTTPSCertificate.js';
import { Buffer } from 'buffer';
import FetchBodyUtility from './utilities/FetchBodyUtility.js';
const LAST_CHUNK = Buffer.from('0\r\n\r\n');
/**
 * Handles fetch requests.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/index.js
 *
 * @see https://fetch.spec.whatwg.org/#http-network-fetch
 */
class Fetch {
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
        this.reject = null;
        this.resolve = null;
        this.listeners = {
            onSignalAbort: this.onSignalAbort.bind(this)
        };
        this.isChunkedTransfer = false;
        this.isProperLastChunkReceived = false;
        this.previousChunk = null;
        this.nodeRequest = null;
        this.nodeResponse = null;
        this.response = null;
        this.responseHeaders = null;
        this.redirectCount = 0;
        _Fetch_browserFrame.set(this, void 0);
        _Fetch_window.set(this, void 0);
        __classPrivateFieldSet(this, _Fetch_browserFrame, options.browserFrame, "f");
        __classPrivateFieldSet(this, _Fetch_window, options.window, "f");
        this.request =
            typeof options.url === 'string' || options.url instanceof URL
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
    async send() {
        FetchRequestReferrerUtility.prepareRequest(new URL(__classPrivateFieldGet(this, _Fetch_window, "f").location.href), this.request);
        FetchRequestValidationUtility.validateSchema(this.request);
        if (this.request.signal.aborted) {
            throw new DOMException('The operation was aborted.', DOMExceptionNameEnum.abortError);
        }
        if (this.request[PropertySymbol.url].protocol === 'data:') {
            const result = DataURIParser.parse(this.request.url);
            this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(result.buffer, {
                headers: { 'Content-Type': result.type }
            });
            return this.response;
        }
        // Security check for "https" to "http" requests.
        if (this.request[PropertySymbol.url].protocol === 'http:' &&
            __classPrivateFieldGet(this, _Fetch_window, "f").location.protocol === 'https:') {
            throw new DOMException(`Mixed Content: The page at '${__classPrivateFieldGet(this, _Fetch_window, "f").location.href}' was loaded over HTTPS, but requested an insecure XMLHttpRequest endpoint '${this.request.url}'. This request has been blocked; the content must be served over HTTPS.`, DOMExceptionNameEnum.securityError);
        }
        if (!this.disableCache) {
            const cachedResponse = await this.getCachedResponse();
            if (cachedResponse) {
                return cachedResponse;
            }
        }
        if (!this.disableCrossOriginPolicy) {
            const compliesWithCrossOriginPolicy = await this.compliesWithCrossOriginPolicy();
            if (!compliesWithCrossOriginPolicy) {
                __classPrivateFieldGet(this, _Fetch_window, "f").console.warn(`Cross-Origin Request Blocked: The Same Origin Policy dissallows reading the remote resource at "${this.request.url}".`);
                throw new DOMException(`Cross-Origin Request Blocked: The Same Origin Policy dissallows reading the remote resource at "${this.request.url}".`, DOMExceptionNameEnum.networkError);
            }
        }
        return await this.sendRequest();
    }
    /**
     * Returns cached response.
     *
     * @returns Response.
     */
    async getCachedResponse() {
        if (this.disableCache) {
            return null;
        }
        let cachedResponse = __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context.responseCache.get(this.request);
        if (!cachedResponse || cachedResponse.response.waitingForBody) {
            return null;
        }
        if (cachedResponse.state === CachedResponseStateEnum.stale) {
            const headers = new Headers(cachedResponse.request.headers);
            if (cachedResponse.etag) {
                headers.set('If-None-Match', cachedResponse.etag);
            }
            else {
                if (!cachedResponse.lastModified) {
                    return null;
                }
                headers.set('If-Modified-Since', new Date(cachedResponse.lastModified).toUTCString());
            }
            const fetch = new Fetch({
                browserFrame: __classPrivateFieldGet(this, _Fetch_browserFrame, "f"),
                window: __classPrivateFieldGet(this, _Fetch_window, "f"),
                url: this.request.url,
                init: { headers, method: cachedResponse.request.method },
                disableCache: true,
                disableCrossOriginPolicy: true
            });
            if (cachedResponse.etag || !cachedResponse.staleWhileRevalidate) {
                const validateResponse = await fetch.send();
                const body = validateResponse.status !== 304 ? await validateResponse.buffer() : null;
                cachedResponse = __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context.responseCache.add(this.request, {
                    ...validateResponse,
                    body,
                    waitingForBody: false
                });
                if (validateResponse.status !== 304) {
                    const response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(body, {
                        status: validateResponse.status,
                        statusText: validateResponse.statusText,
                        headers: validateResponse.headers
                    });
                    response.url = validateResponse.url;
                    response[PropertySymbol.cachedResponse] = cachedResponse;
                    return response;
                }
            }
            else {
                fetch.send().then((response) => {
                    response.buffer().then((body) => {
                        __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context.responseCache.add(this.request, {
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
        const response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(cachedResponse.response.body, {
            status: cachedResponse.response.status,
            statusText: cachedResponse.response.statusText,
            headers: cachedResponse.response.headers
        });
        response.url = cachedResponse.response.url;
        response[PropertySymbol.cachedResponse] = cachedResponse;
        return response;
    }
    /**
     * Checks if the request complies with the Cross-Origin policy.
     *
     * @returns True if it complies with the policy.
     */
    async compliesWithCrossOriginPolicy() {
        if (this.disableCrossOriginPolicy ||
            !FetchCORSUtility.isCORS(__classPrivateFieldGet(this, _Fetch_window, "f").location.href, this.request[PropertySymbol.url])) {
            return true;
        }
        const cachedPreflightResponse = __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context.preflightResponseCache.get(this.request);
        if (cachedPreflightResponse) {
            if (cachedPreflightResponse.allowOrigin !== '*' &&
                cachedPreflightResponse.allowOrigin !== __classPrivateFieldGet(this, _Fetch_window, "f").location.origin) {
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
        const fetch = new Fetch({
            browserFrame: __classPrivateFieldGet(this, _Fetch_browserFrame, "f"),
            window: __classPrivateFieldGet(this, _Fetch_window, "f"),
            url: this.request.url,
            init: {
                method: 'OPTIONS',
                headers: new Headers({
                    'Access-Control-Request-Method': this.request.method,
                    'Access-Control-Request-Headers': requestHeaders.join(', ')
                })
            },
            disableCache: true,
            disableCrossOriginPolicy: true
        });
        const response = await fetch.send();
        if (!response.ok) {
            return false;
        }
        const allowOrigin = response.headers.get('Access-Control-Allow-Origin');
        if (!allowOrigin) {
            return false;
        }
        if (allowOrigin !== '*' && allowOrigin !== __classPrivateFieldGet(this, _Fetch_window, "f").location.origin) {
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
        return new Promise((resolve, reject) => {
            const taskID = __classPrivateFieldGet(this, _Fetch_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask(() => this.onAsyncTaskManagerAbort());
            if (this.resolve) {
                throw new Error('Fetch already sent.');
            }
            this.resolve = (response) => {
                // We can end up here when closing down the browser frame and there is an ongoing request.
                // Therefore we need to check if browserFrame.page.context is still available.
                if (!this.disableCache &&
                    response instanceof Response &&
                    __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page &&
                    __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context) {
                    response[PropertySymbol.cachedResponse] =
                        __classPrivateFieldGet(this, _Fetch_browserFrame, "f").page.context.responseCache.add(this.request, {
                            ...response,
                            headers: this.responseHeaders,
                            body: response[PropertySymbol.buffer],
                            waitingForBody: !response[PropertySymbol.buffer] && !!response.body
                        });
                }
                __classPrivateFieldGet(this, _Fetch_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
                resolve(response);
            };
            this.reject = (error) => {
                __classPrivateFieldGet(this, _Fetch_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
                reject(error);
            };
            this.request.signal.addEventListener('abort', this.listeners.onSignalAbort);
            const send = (this.request[PropertySymbol.url].protocol === 'https:' ? HTTPS : HTTP).request;
            this.nodeRequest = send(this.request[PropertySymbol.url].href, {
                method: this.request.method,
                headers: FetchRequestHeaderUtility.getRequestHeaders({
                    browserFrame: __classPrivateFieldGet(this, _Fetch_browserFrame, "f"),
                    window: __classPrivateFieldGet(this, _Fetch_window, "f"),
                    request: this.request
                }),
                agent: false,
                rejectUnauthorized: true,
                key: this.request[PropertySymbol.url].protocol === 'https:'
                    ? FetchHTTPSCertificate.key
                    : undefined,
                cert: this.request[PropertySymbol.url].protocol === 'https:'
                    ? FetchHTTPSCertificate.cert
                    : undefined
            });
            this.nodeRequest.on('error', this.onError.bind(this));
            this.nodeRequest.on('socket', this.onSocket.bind(this));
            this.nodeRequest.on('response', this.onResponse.bind(this));
            if (this.request.body === null) {
                this.nodeRequest.end();
            }
            else {
                Stream.pipeline(this.request.body, this.nodeRequest, (error) => {
                    if (error) {
                        this.onError(error);
                    }
                });
            }
        });
    }
    /**
     * Event listener for "socket" event.
     *
     * @param socket Socket.
     */
    onSocket(socket) {
        const onSocketClose = () => {
            if (this.isChunkedTransfer && !this.isProperLastChunkReceived) {
                const error = new DOMException('Premature close.', DOMExceptionNameEnum.networkError);
                if (this.response && this.response.body) {
                    this.response.body[PropertySymbol.error] = error;
                    if (!this.response.body.locked) {
                        this.response.body.cancel(error);
                    }
                }
            }
        };
        const onData = (buffer) => {
            this.isProperLastChunkReceived = Buffer.compare(buffer.slice(-5), LAST_CHUNK) === 0;
            // Sometimes final 0-length chunk and end of message code are in separate packets.
            if (!this.isProperLastChunkReceived && this.previousChunk) {
                this.isProperLastChunkReceived =
                    Buffer.compare(this.previousChunk.slice(-3), LAST_CHUNK.slice(0, 3)) === 0 &&
                        Buffer.compare(buffer.slice(-2), LAST_CHUNK.slice(3)) === 0;
            }
            this.previousChunk = buffer;
        };
        socket.prependListener('close', onSocketClose);
        socket.on('data', onData);
        this.nodeRequest.on('close', () => {
            socket.removeListener('close', onSocketClose);
            socket.removeListener('data', onData);
        });
    }
    /**
     * Event listener for signal "abort" event.
     *
     * @param event Event.
     */
    onSignalAbort(event) {
        this.finalizeRequest();
        this.abort(event.target?.reason);
    }
    /**
     * Event listener for request "error" event.
     *
     * @param error Error.
     */
    onError(error) {
        this.finalizeRequest();
        __classPrivateFieldGet(this, _Fetch_window, "f").console.error(error);
        this.reject(new DOMException(`Fetch to "${this.request.url}" failed. Error: ${error.message}`, DOMExceptionNameEnum.networkError));
    }
    /**
     * Triggered when the async task manager aborts.
     */
    onAsyncTaskManagerAbort() {
        const error = new DOMException('The operation was aborted.', DOMExceptionNameEnum.abortError);
        if (this.nodeRequest && !this.nodeRequest.destroyed) {
            this.nodeRequest.destroy(error);
        }
        if (this.nodeResponse && !this.nodeResponse.destroyed) {
            this.nodeResponse.destroy(error);
        }
        if (this.response && this.response.body) {
            this.response.body[PropertySymbol.error] = error;
            if (!this.response.body.locked) {
                this.response.body.cancel(error);
            }
        }
    }
    /**
     * Event listener for request "response" event.
     *
     * @param nodeResponse Node response.
     */
    onResponse(nodeResponse) {
        // Needed for handling bad endings of chunked transfer.
        this.isChunkedTransfer =
            nodeResponse.headers['transfer-encoding'] === 'chunked' &&
                !nodeResponse.headers['content-length'];
        this.nodeRequest.setTimeout(0);
        this.responseHeaders = FetchResponseHeaderUtility.parseResponseHeaders({
            browserFrame: __classPrivateFieldGet(this, _Fetch_browserFrame, "f"),
            requestURL: this.request[PropertySymbol.url],
            rawHeaders: nodeResponse.rawHeaders
        });
        if (this.handleRedirectResponse(nodeResponse, this.responseHeaders)) {
            return;
        }
        nodeResponse.once('end', () => this.request.signal.removeEventListener('abort', this.listeners.onSignalAbort));
        let body = Stream.pipeline(nodeResponse, new Stream.PassThrough(), (error) => {
            if (error) {
                // Ignore error as it is forwarded to the response body.
            }
        });
        const responseOptions = {
            status: nodeResponse.statusCode,
            statusText: nodeResponse.statusMessage,
            headers: this.responseHeaders
        };
        const contentEncodingHeader = this.responseHeaders.get('Content-Encoding');
        if (this.request.method === 'HEAD' ||
            contentEncodingHeader === null ||
            nodeResponse.statusCode === 204 ||
            nodeResponse.statusCode === 304) {
            this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
            this.response.redirected = this.redirectCount > 0;
            this.response.url = this.request.url;
            this.resolve(this.response);
            return;
        }
        // For GZip
        if (contentEncodingHeader === 'gzip' || contentEncodingHeader === 'x-gzip') {
            // Be less strict when decoding compressed responses.
            // Sometimes servers send slightly invalid responses that are still accepted by common browsers.
            // "cURL" always uses Z_SYNC_FLUSH.
            const zlibOptions = {
                flush: Zlib.constants.Z_SYNC_FLUSH,
                finishFlush: Zlib.constants.Z_SYNC_FLUSH
            };
            body = Stream.pipeline(body, Zlib.createGunzip(zlibOptions), (error) => {
                if (error) {
                    // Ignore error as it is forwarded to the response body.
                }
            });
            this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
            this.response.redirected = this.redirectCount > 0;
            this.response.url = this.request.url;
            this.resolve(this.response);
            return;
        }
        // For Deflate
        if (contentEncodingHeader === 'deflate' || contentEncodingHeader === 'x-deflate') {
            // Handle the infamous raw deflate response from old servers
            // A hack for old IIS and Apache servers
            const raw = Stream.pipeline(nodeResponse, new Stream.PassThrough(), (error) => {
                if (error) {
                    // Ignore error as it is forwarded to the response body.
                }
            });
            raw.on('data', (chunk) => {
                // See http://stackoverflow.com/questions/37519828
                if ((chunk[0] & 0x0f) === 0x08) {
                    body = Stream.pipeline(body, Zlib.createInflate(), (error) => {
                        if (error) {
                            // Ignore error as the fetch() promise has already been resolved.
                        }
                    });
                }
                else {
                    body = Stream.pipeline(body, Zlib.createInflateRaw(), (error) => {
                        if (error) {
                            // Ignore error as it is forwarded to the response body.
                        }
                    });
                }
                this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
                this.response.redirected = this.redirectCount > 0;
                this.response.url = this.request.url;
                this.resolve(this.response);
            });
            raw.on('end', () => {
                // Some old IIS servers return zero-length OK deflate responses, so 'data' is never emitted.
                if (!this.response) {
                    this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
                    this.response.redirected = this.redirectCount > 0;
                    this.response.url = this.request.url;
                    this.resolve(this.response);
                }
            });
            return;
        }
        // For BR
        if (contentEncodingHeader === 'br') {
            body = Stream.pipeline(body, Zlib.createBrotliDecompress(), (error) => {
                if (error) {
                    // Ignore error as it is forwarded to the response body.
                }
            });
            this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
            this.response.redirected = this.redirectCount > 0;
            this.response.url = this.request.url;
            this.resolve(this.response);
            return;
        }
        // Otherwise, use response as is
        this.response = new (__classPrivateFieldGet(this, _Fetch_window, "f").Response)(FetchBodyUtility.nodeToWebStream(body), responseOptions);
        this.response.redirected = this.redirectCount > 0;
        this.response.url = this.request.url;
        this.resolve(this.response);
    }
    /**
     * Handles redirect response.
     *
     * @param nodeResponse Node response.
     * @param responseHeaders Headers.
     * @returns True if redirect response was handled, false otherwise.
     */
    handleRedirectResponse(nodeResponse, responseHeaders) {
        if (!FetchResponseRedirectUtility.isRedirect(nodeResponse.statusCode)) {
            return false;
        }
        switch (this.request.redirect) {
            case 'error':
                this.finalizeRequest();
                this.reject(new DOMException(`URI requested responds with a redirect, redirect mode is set to "error": ${this.request.url}`, DOMExceptionNameEnum.abortError));
                return true;
            case 'manual':
                // Nothing to do
                return false;
            case 'follow':
                const locationHeader = responseHeaders.get('Location');
                const shouldBecomeGetRequest = nodeResponse.statusCode === 303 ||
                    ((nodeResponse.statusCode === 301 || nodeResponse.statusCode === 302) &&
                        this.request.method === 'POST');
                let locationURL = null;
                if (locationHeader !== null) {
                    try {
                        locationURL = new URL(locationHeader, this.request.url);
                    }
                    catch {
                        this.finalizeRequest();
                        this.reject(new DOMException(`URI requested responds with an invalid redirect URL: ${locationHeader}`, DOMExceptionNameEnum.uriMismatchError));
                        return true;
                    }
                }
                if (locationURL === null) {
                    return false;
                }
                if (FetchResponseRedirectUtility.isMaxRedirectsReached(this.redirectCount)) {
                    this.finalizeRequest();
                    this.reject(new DOMException(`Maximum redirects reached at: ${this.request.url}`, DOMExceptionNameEnum.networkError));
                    return true;
                }
                const headers = new Headers(this.request.headers);
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
                        FetchCORSUtility.isCORS(__classPrivateFieldGet(this, _Fetch_window, "f").location.href, locationURL))) {
                    headers.delete('authorization');
                    headers.delete('www-authenticate');
                    headers.delete('cookie');
                    headers.delete('cookie2');
                }
                if (this.request.signal.aborted) {
                    this.abort();
                    return true;
                }
                if (shouldBecomeGetRequest) {
                    requestInit.method = 'GET';
                    requestInit.body = undefined;
                    headers.delete('Content-Length');
                    headers.delete('Content-Type');
                }
                const responseReferrerPolicy = FetchRequestReferrerUtility.getReferrerPolicyFromHeader(headers);
                if (responseReferrerPolicy) {
                    requestInit.referrerPolicy = responseReferrerPolicy;
                }
                const fetch = new Fetch({
                    browserFrame: __classPrivateFieldGet(this, _Fetch_browserFrame, "f"),
                    window: __classPrivateFieldGet(this, _Fetch_window, "f"),
                    url: locationURL,
                    init: requestInit,
                    redirectCount: this.redirectCount + 1,
                    contentType: !shouldBecomeGetRequest
                        ? this.request[PropertySymbol.contentType]
                        : undefined
                });
                this.finalizeRequest();
                fetch
                    .send()
                    .then((response) => this.resolve(response))
                    .catch((error) => this.reject(error));
                return true;
            default:
                this.finalizeRequest();
                this.reject(new DOMException(`Redirect option '${this.request.redirect}' is not a valid value of IRequestRedirect`));
                return true;
        }
    }
    /**
     * Finalizes the request.
     */
    finalizeRequest() {
        this.request.signal.removeEventListener('abort', this.listeners.onSignalAbort);
        this.nodeRequest.destroy();
    }
    /**
     * Aborts the request.
     *
     * @param reason Reason.
     */
    abort(reason) {
        const error = new DOMException('The operation was aborted.' + (reason ? ' ' + reason.toString() : ''), DOMExceptionNameEnum.abortError);
        if (this.nodeRequest && !this.nodeRequest.destroyed) {
            this.nodeRequest.destroy(error);
        }
        if (this.nodeResponse && !this.nodeResponse.destroyed) {
            this.nodeResponse.destroy(error);
        }
        if (this.response && this.response.body) {
            this.response.body[PropertySymbol.error] = error;
            if (!this.response.body.locked) {
                this.response.body.cancel(error);
            }
        }
        if (this.reject) {
            this.reject(error);
        }
    }
}
_Fetch_browserFrame = new WeakMap(), _Fetch_window = new WeakMap();
export default Fetch;
//# sourceMappingURL=Fetch.js.map