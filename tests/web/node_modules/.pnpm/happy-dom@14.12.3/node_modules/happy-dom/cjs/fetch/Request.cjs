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
var _Request_window, _Request_asyncTaskManager, _a, _b, _c;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const Headers_js_1 = __importDefault(require("./Headers.cjs"));
const FetchBodyUtility_js_1 = __importDefault(require("./utilities/FetchBodyUtility.cjs"));
const AbortSignal_js_1 = __importDefault(require("./AbortSignal.cjs"));
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
const FetchRequestValidationUtility_js_1 = __importDefault(require("./utilities/FetchRequestValidationUtility.cjs"));
const FetchRequestReferrerUtility_js_1 = __importDefault(require("./utilities/FetchRequestReferrerUtility.cjs"));
const FetchRequestHeaderUtility_js_1 = __importDefault(require("./utilities/FetchRequestHeaderUtility.cjs"));
const FormData_js_1 = __importDefault(require("../form-data/FormData.cjs"));
const MultipartFormDataParser_js_1 = __importDefault(require("./multipart/MultipartFormDataParser.cjs"));
/**
 * Fetch request.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/request.js
 *
 * @see https://fetch.spec.whatwg.org/#request-class
 */
class Request {
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
        const { stream, buffer, contentType, contentLength } = FetchBodyUtility_js_1.default.getBodyStream(input instanceof Request && (input[PropertySymbol.bodyBuffer] || input.body)
            ? input[PropertySymbol.bodyBuffer] || FetchBodyUtility_js_1.default.cloneBodyStream(input)
            : init?.body);
        this[PropertySymbol.bodyBuffer] = buffer;
        this.body = stream;
        this.credentials = init?.credentials || input.credentials || 'same-origin';
        this.headers = new Headers_js_1.default(init?.headers || input.headers || {});
        FetchRequestHeaderUtility_js_1.default.removeForbiddenHeaders(this.headers);
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
        this.signal = init?.signal || input.signal || new AbortSignal_js_1.default();
        this[PropertySymbol.referrer] = FetchRequestReferrerUtility_js_1.default.getInitialReferrer(injected.window, init?.referrer !== null && init?.referrer !== undefined
            ? init?.referrer
            : input.referrer);
        if (input instanceof URL_js_1.default) {
            this[PropertySymbol.url] = input;
        }
        else {
            try {
                if (input instanceof Request && input.url) {
                    this[PropertySymbol.url] = new URL_js_1.default(input.url, injected.window.location.href);
                }
                else {
                    this[PropertySymbol.url] = new URL_js_1.default(input, injected.window.location.href);
                }
            }
            catch (error) {
                throw new DOMException_js_1.default(`Failed to construct 'Request. Invalid URL "${input}" on document location '${injected.window.location}'.${injected.window.location.origin === 'null'
                    ? ' Relative URLs are not permitted on current document location.'
                    : ''}`, DOMExceptionNameEnum_js_1.default.notSupportedError);
            }
        }
        FetchRequestValidationUtility_js_1.default.validateMethod(this);
        FetchRequestValidationUtility_js_1.default.validateBody(this);
        FetchRequestValidationUtility_js_1.default.validateURL(this[PropertySymbol.url]);
        FetchRequestValidationUtility_js_1.default.validateReferrerPolicy(this.referrerPolicy);
        FetchRequestValidationUtility_js_1.default.validateRedirect(this.redirect);
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
            throw new DOMException_js_1.default(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
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
        return new Blob_js_1.default([buffer], { type });
    }
    /**
     * Returns buffer.
     *
     * @returns Buffer.
     */
    async buffer() {
        if (this.bodyUsed) {
            throw new DOMException_js_1.default(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
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
            throw new DOMException_js_1.default(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        this.bodyUsed = true;
        const taskID = __classPrivateFieldGet(this, _Request_asyncTaskManager, "f").startTask(() => this.signal[PropertySymbol.abort]());
        let buffer;
        try {
            buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
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
                throw new DOMException_js_1.default(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
            }
            this.bodyUsed = true;
            const taskID = asyncTaskManager.startTask(() => this.signal[PropertySymbol.abort]());
            let formData;
            try {
                const result = await MultipartFormDataParser_js_1.default.streamToFormData(this.body, contentType);
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
            const formData = new FormData_js_1.default();
            for (const [key, value] of parameters) {
                formData.append(key, value);
            }
            return formData;
        }
        throw new DOMException_js_1.default(`Failed to construct FormData object: The "content-type" header is neither "application/x-www-form-urlencoded" nor "multipart/form-data".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
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
exports.default = Request;
//# sourceMappingURL=Request.cjs.map