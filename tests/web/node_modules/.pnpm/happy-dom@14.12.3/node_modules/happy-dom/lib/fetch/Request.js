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
var _Request_window, _Request_asyncTaskManager, _a, _b, _c;
import * as PropertySymbol from '../PropertySymbol.js';
import URL from '../url/URL.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import Headers from './Headers.js';
import FetchBodyUtility from './utilities/FetchBodyUtility.js';
import AbortSignal from './AbortSignal.js';
import Blob from '../file/Blob.js';
import FetchRequestValidationUtility from './utilities/FetchRequestValidationUtility.js';
import FetchRequestReferrerUtility from './utilities/FetchRequestReferrerUtility.js';
import FetchRequestHeaderUtility from './utilities/FetchRequestHeaderUtility.js';
import FormData from '../form-data/FormData.js';
import MultipartFormDataParser from './multipart/MultipartFormDataParser.js';
/**
 * Fetch request.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/request.js
 *
 * @see https://fetch.spec.whatwg.org/#request-class
 */
export default class Request {
    /**
     * Constructor.
     *
     * @param injected Injected properties.
     * @param injected.window
     * @param input Input.
     * @param injected.asyncTaskManager
     * @param [init] Init.
     */
    constructor(injected, input, init) {
        this.bodyUsed = false;
        // Internal properties
        this[_a] = null;
        this[_b] = null;
        this[_c] = 'client';
        // Private properties
        _Request_window.set(this, void 0);
        _Request_asyncTaskManager.set(this, void 0);
        __classPrivateFieldSet(this, _Request_window, injected.window, "f");
        __classPrivateFieldSet(this, _Request_asyncTaskManager, injected.asyncTaskManager, "f");
        if (!input) {
            throw new TypeError(`Failed to contruct 'Request': 1 argument required, only 0 present.`);
        }
        this.method = (init?.method || input.method || 'GET').toUpperCase();
        const { stream, buffer, contentType, contentLength } = FetchBodyUtility.getBodyStream(input instanceof Request && (input[PropertySymbol.bodyBuffer] || input.body)
            ? input[PropertySymbol.bodyBuffer] || FetchBodyUtility.cloneBodyStream(input)
            : init?.body);
        this[PropertySymbol.bodyBuffer] = buffer;
        this.body = stream;
        this.credentials = init?.credentials || input.credentials || 'same-origin';
        this.headers = new Headers(init?.headers || input.headers || {});
        FetchRequestHeaderUtility.removeForbiddenHeaders(this.headers);
        if (contentLength) {
            this[PropertySymbol.contentLength] = contentLength;
        }
        else if (!this.body && (this.method === 'POST' || this.method === 'PUT')) {
            this[PropertySymbol.contentLength] = 0;
        }
        if (contentType) {
            if (!this.headers.has('Content-Type')) {
                this.headers.set('Content-Type', contentType);
            }
            this[PropertySymbol.contentType] = contentType;
        }
        else if (input instanceof Request && input[PropertySymbol.contentType]) {
            this[PropertySymbol.contentType] = input[PropertySymbol.contentType];
        }
        this.redirect = init?.redirect || input.redirect || 'follow';
        this.referrerPolicy = ((init?.referrerPolicy || input.referrerPolicy || '').toLowerCase());
        this.signal = init?.signal || input.signal || new AbortSignal();
        this[PropertySymbol.referrer] = FetchRequestReferrerUtility.getInitialReferrer(injected.window, init?.referrer !== null && init?.referrer !== undefined
            ? init?.referrer
            : input.referrer);
        if (input instanceof URL) {
            this[PropertySymbol.url] = input;
        }
        else {
            try {
                if (input instanceof Request && input.url) {
                    this[PropertySymbol.url] = new URL(input.url, injected.window.location.href);
                }
                else {
                    this[PropertySymbol.url] = new URL(input, injected.window.location.href);
                }
            }
            catch (error) {
                throw new DOMException(`Failed to construct 'Request. Invalid URL "${input}" on document location '${injected.window.location}'.${injected.window.location.origin === 'null'
                    ? ' Relative URLs are not permitted on current document location.'
                    : ''}`, DOMExceptionNameEnum.notSupportedError);
            }
        }
        FetchRequestValidationUtility.validateMethod(this);
        FetchRequestValidationUtility.validateBody(this);
        FetchRequestValidationUtility.validateURL(this[PropertySymbol.url]);
        FetchRequestValidationUtility.validateReferrerPolicy(this.referrerPolicy);
        FetchRequestValidationUtility.validateRedirect(this.redirect);
    }
    /**
     * Returns owner document.
     */
    get [(_Request_window = new WeakMap(), _Request_asyncTaskManager = new WeakMap(), _a = PropertySymbol.contentLength, _b = PropertySymbol.contentType, _c = PropertySymbol.referrer, PropertySymbol.url, PropertySymbol.bodyBuffer, PropertySymbol.ownerDocument)]() {
        throw new Error('[PropertySymbol.ownerDocument] getter needs to be implemented by sub-class.');
    }
    /**
     * Returns referrer.
     *
     * @returns Referrer.
     */
    get referrer() {
        if (!this[PropertySymbol.referrer] || this[PropertySymbol.referrer] === 'no-referrer') {
            return '';
        }
        if (this[PropertySymbol.referrer] === 'client') {
            return 'about:client';
        }
        return this[PropertySymbol.referrer].toString();
    }
    /**
     * Returns URL.
     *
     * @returns URL.
     */
    get url() {
        return this[PropertySymbol.url].href;
    }
    /**
     * Returns string tag.
     *
     * @returns String tag.
     */
    get [Symbol.toStringTag]() {
        return 'Request';
    }
    /**
     * Returns array buffer.
     *
     * @returns Array buffer.
     */
    async arrayBuffer() {
        if (this.bodyUsed) {
            throw new DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility.consumeBodyStream(this.body);
        }
        catch (error) {
            __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
            throw error;
        }
        __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
        return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
    }
    /**
     * Returns blob.
     *
     * @returns Blob.
     */
    async blob() {
        const type = this.headers.get('Content-Type') || '';
        const buffer = await this.arrayBuffer();
        return new Blob([buffer], { type });
    }
    /**
     * Returns buffer.
     *
     * @returns Buffer.
     */
    async buffer() {
        if (this.bodyUsed) {
            throw new DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility.consumeBodyStream(this.body);
        }
        catch (error) {
            __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
            throw error;
        }
        __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
        return buffer;
    }
    /**
     * Returns text.
     *
     * @returns Text.
     */
    async text() {
        if (this.bodyUsed) {
            throw new DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility.consumeBodyStream(this.body);
        }
        catch (error) {
            __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
            throw error;
        }
        __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").endTask(taskID);
        return new TextDecoder().decode(buffer);
    }
    /**
     * Returns json.
     *
     * @returns JSON.
     */
    async json() {
        const text = await this.text();
        return JSON.parse(text);
    }
    /**
     * Returns FormData.
     *
     * @returns FormData.
     */
    async formData() {
        const contentType = this[PropertySymbol.contentType];
        const asyncTaskManager = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f");
        if (/multipart/i.test(contentType)) {
            if (this.bodyUsed) {
                throw new DOMException(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum.invalidStateError);
            }
            this.bodyUsed = true;
            const taskID = asyncTaskManager.startTask(() => this.signal[PropertySymbol.abort]());
            let formData;
            try {
                const result = await MultipartFormDataParser.streamToFormData(this.body, contentType);
                formData = result.formData;
            }
            catch (error) {
                asyncTaskManager.endTask(taskID);
                throw error;
            }
            asyncTaskManager.endTask(taskID);
            return formData;
        }
        if (contentType?.startsWith('application/x-www-form-urlencoded')) {
            const parameters = new URLSearchParams(await this.text());
            const formData = new FormData();
            for (const [key, value] of parameters) {
                formData.append(key, value);
            }
            return formData;
        }
        throw new DOMException(`Failed to construct FormData object: The "content-type" header is neither "application/x-www-form-urlencoded" nor "multipart/form-data".`, DOMExceptionNameEnum.invalidStateError);
    }
    /**
     * Clones request.
     *
     * @returns Clone.
     */
    clone() {
        return new (__classPrivateFieldGet(this, _Request_window, "f").Request)(this);
    }
}
//# sourceMappingURL=Request.js.map