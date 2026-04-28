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
var _Response_instances, _Response_window, _Response_browserFrame, _Response_storeBodyInCache, _a, _b;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
const Headers_js_1 = __importDefault(require("./Headers.cjs"));
const url_1 = require("url");
const URL_js_1 = __importDefault(require("../url/URL.cjs"));
const FormData_js_1 = __importDefault(require("../form-data/FormData.cjs"));
const FetchBodyUtility_js_1 = __importDefault(require("./utilities/FetchBodyUtility.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const MultipartFormDataParser_js_1 = __importDefault(require("./multipart/MultipartFormDataParser.cjs"));
const REDIRECT_STATUS_CODES = [301, 302, 303, 307, 308];
/**
 * Fetch response.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/response.js (MIT)
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Response/Response
 */
class Response {
    /**
     * Constructor.
     *
     * @param injected Injected properties.
     * @param input Input.
     * @param injected.window
     * @param body
     * @param injected.browserFrame
     * @param [init] Init.
     */
    constructor(injected, body, init) {
        _Response_instances.add(this);
        // Public properties
        this.body = null;
        this.bodyUsed = false;
        this.redirected = false;
        this.type = 'basic';
        this.url = '';
        this[_a] = null;
        this[_b] = null;
        _Response_window.set(this, void 0);
        _Response_browserFrame.set(this, void 0);
        __classPrivateFieldSet(this, _Response_window, injected.window, "f");
        __classPrivateFieldSet(this, _Response_browserFrame, injected.browserFrame, "f");
        this.status = init?.status !== undefined ? init.status : 200;
        this.statusText = init?.statusText || '';
        this.ok = this.status >= 200 && this.status < 300;
        this.headers = new Headers_js_1.default(init?.headers);
        // "Set-Cookie" and "Set-Cookie2" are not allowed in response headers according to spec.
        this.headers.delete('Set-Cookie');
        this.headers.delete('Set-Cookie2');
        if (body) {
            const { stream, buffer, contentType } = FetchBodyUtility_js_1.default.getBodyStream(body);
            this.body = stream;
            if (buffer) {
                this[PropertySymbol.buffer] = buffer;
            }
            if (contentType && !this.headers.has('Content-Type')) {
                this.headers.set('Content-Type', contentType);
            }
        }
    }
    /**
     * Returns string tag.
     *
     * @returns String tag.
     */
    get [(_Response_window = new WeakMap(), _Response_browserFrame = new WeakMap(), _Response_instances = new WeakSet(), PropertySymbol.window, _a = PropertySymbol.cachedResponse, _b = PropertySymbol.buffer, Symbol.toStringTag)]() {
        return 'Response';
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
        let buffer = this[PropertySymbol.buffer];
        if (!buffer) {
            const taskID = __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask();
            try {
                buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
            }
            catch (error) {
                __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
                throw error;
            }
            __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
        }
        __classPrivateFieldGet(this, _Response_instances, "m", _Response_storeBodyInCache).call(this, buffer);
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
        let buffer = this[PropertySymbol.buffer];
        if (!buffer) {
            const taskID = __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask();
            try {
                buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
            }
            catch (error) {
                __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
                throw error;
            }
            __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
        }
        __classPrivateFieldGet(this, _Response_instances, "m", _Response_storeBodyInCache).call(this, buffer);
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
        let buffer = this[PropertySymbol.buffer];
        if (!buffer) {
            const taskID = __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask();
            try {
                buffer = await FetchBodyUtility_js_1.default.consumeBodyStream(this.body);
            }
            catch (error) {
                __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
                throw error;
            }
            __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
        }
        __classPrivateFieldGet(this, _Response_instances, "m", _Response_storeBodyInCache).call(this, buffer);
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
     * Returns form data.
     *
     * @returns Form data.
     */
    async formData() {
        const contentType = this.headers.get('Content-Type');
        const asyncTaskManager = __classPrivateFieldGet(this, _Response_browserFrame, "f")[PropertySymbol.asyncTaskManager];
        if (/multipart/i.test(contentType)) {
            if (this.bodyUsed) {
                throw new DOMException_js_1.default(`Body has already been used for "${this.url}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
            }
            this.bodyUsed = true;
            const taskID = asyncTaskManager.startTask();
            let formData;
            let buffer;
            try {
                const result = await MultipartFormDataParser_js_1.default.streamToFormData(this.body, contentType);
                formData = result.formData;
                buffer = result.buffer;
            }
            catch (error) {
                asyncTaskManager.endTask(taskID);
                throw error;
            }
            __classPrivateFieldGet(this, _Response_instances, "m", _Response_storeBodyInCache).call(this, buffer);
            asyncTaskManager.endTask(taskID);
            return formData;
        }
        if (contentType?.startsWith('application/x-www-form-urlencoded')) {
            const parameters = new url_1.URLSearchParams(await this.text());
            const formData = new FormData_js_1.default();
            for (const [key, value] of parameters) {
                formData.append(key, value);
            }
            return formData;
        }
        throw new DOMException_js_1.default(`Failed to build FormData object: The "content-type" header is neither "application/x-www-form-urlencoded" nor "multipart/form-data".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
    }
    /**
     * Clones request.
     *
     * @returns Clone.
     */
    clone() {
        const body = FetchBodyUtility_js_1.default.cloneBodyStream(this);
        const response = new (__classPrivateFieldGet(this, _Response_window, "f").Response)(body, {
            status: this.status,
            statusText: this.statusText,
            headers: this.headers
        });
        response[PropertySymbol.cachedResponse] = this[PropertySymbol.cachedResponse];
        response[PropertySymbol.buffer] = this[PropertySymbol.buffer];
        response.ok = this.ok;
        response.redirected = this.redirected;
        response.type = this.type;
        response.url = this.url;
        return response;
    }
    /**
     * Returns a redirect response.
     *
     * @param url URL.
     * @param status Status code.
     * @returns Response.
     */
    static redirect(url, status = 302) {
        if (!REDIRECT_STATUS_CODES.includes(status)) {
            throw new DOMException_js_1.default('Failed to create redirect response: Invalid redirect status code.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        return new this[PropertySymbol.window].Response(null, {
            headers: {
                location: new URL_js_1.default(url).toString()
            },
            status
        });
    }
    /**
     * Returns an error response.
     *
     * @param url URL.
     * @param status Status code.
     * @returns Response.
     */
    static error() {
        const response = new this[PropertySymbol.window].Response(null, { status: 0, statusText: '' });
        response.type = 'error';
        return response;
    }
    /**
     * Returns an JSON response.
     *
     * @param injected Injected properties.
     * @param data Data.
     * @param [init] Init.
     * @returns Response.
     */
    static json(data, init) {
        const body = JSON.stringify(data);
        if (body === undefined) {
            throw new TypeError('data is not JSON serializable');
        }
        const headers = new this[PropertySymbol.window].Headers(init && init.headers);
        if (!headers.has('Content-Type')) {
            headers.set('Content-Type', 'application/json');
        }
        return new this[PropertySymbol.window].Response(body, {
            status: 200,
            ...init,
            headers
        });
    }
}
_Response_storeBodyInCache = function _Response_storeBodyInCache(buffer) {
    if (this[PropertySymbol.cachedResponse]?.response?.waitingForBody) {
        this[PropertySymbol.cachedResponse].response.body = buffer;
        this[PropertySymbol.cachedResponse].response.waitingForBody = false;
    }
};
exports.default = Response;
//# sourceMappingURL=Response.cjs.map