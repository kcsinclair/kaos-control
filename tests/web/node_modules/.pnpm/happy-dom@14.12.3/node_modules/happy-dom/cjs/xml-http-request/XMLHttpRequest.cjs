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
var _XMLHttpRequest_instances, _XMLHttpRequest_browserFrame, _XMLHttpRequest_window, _XMLHttpRequest_async, _XMLHttpRequest_abortController, _XMLHttpRequest_aborted, _XMLHttpRequest_request, _XMLHttpRequest_response, _XMLHttpRequest_responseType, _XMLHttpRequest_responseBody, _XMLHttpRequest_readyState, _XMLHttpRequest_sendAsync, _XMLHttpRequest_sendSync;
Object.defineProperty(exports, "__esModule", { value: true });
const XMLHttpRequestEventTarget_js_1 = __importDefault(require("./XMLHttpRequestEventTarget.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const XMLHttpRequestReadyStateEnum_js_1 = __importDefault(require("./XMLHttpRequestReadyStateEnum.cjs"));
const Event_js_1 = __importDefault(require("../event/Event.cjs"));
const XMLHttpRequestUpload_js_1 = __importDefault(require("./XMLHttpRequestUpload.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const XMLHttpResponseTypeEnum_js_1 = __importDefault(require("./XMLHttpResponseTypeEnum.cjs"));
const ErrorEvent_js_1 = __importDefault(require("../event/events/ErrorEvent.cjs"));
const Headers_js_1 = __importDefault(require("../fetch/Headers.cjs"));
const Fetch_js_1 = __importDefault(require("../fetch/Fetch.cjs"));
const SyncFetch_js_1 = __importDefault(require("../fetch/SyncFetch.cjs"));
const AbortController_js_1 = __importDefault(require("../fetch/AbortController.cjs"));
const ProgressEvent_js_1 = __importDefault(require("../event/events/ProgressEvent.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../nodes/node/NodeTypeEnum.cjs"));
const XMLHttpRequestResponseDataParser_js_1 = __importDefault(require("./XMLHttpRequestResponseDataParser.cjs"));
const FetchRequestHeaderUtility_js_1 = __importDefault(require("../fetch/utilities/FetchRequestHeaderUtility.cjs"));
/**
 * XMLHttpRequest.
 *
 * Based on:
 * https://github.com/mjwwit/node-XMLHttpRequest/blob/master/lib/XMLHttpRequest.js
 */
class XMLHttpRequest extends XMLHttpRequestEventTarget_js_1.default {
    /**
     * Constructor.
     *
     * @param injected Injected properties.
     * @param injected.browserFrame Browser frame.
     * @param injected.window Window.
     */
    constructor(injected) {
        super();
        _XMLHttpRequest_instances.add(this);
        // Public properties
        this.upload = new XMLHttpRequestUpload_js_1.default();
        this.withCredentials = false;
        // Private properties
        _XMLHttpRequest_browserFrame.set(this, void 0);
        _XMLHttpRequest_window.set(this, void 0);
        _XMLHttpRequest_async.set(this, true);
        _XMLHttpRequest_abortController.set(this, null);
        _XMLHttpRequest_aborted.set(this, false);
        _XMLHttpRequest_request.set(this, null);
        _XMLHttpRequest_response.set(this, null);
        _XMLHttpRequest_responseType.set(this, '');
        _XMLHttpRequest_responseBody.set(this, null);
        _XMLHttpRequest_readyState.set(this, XMLHttpRequestReadyStateEnum_js_1.default.unsent);
        __classPrivateFieldSet(this, _XMLHttpRequest_browserFrame, injected.browserFrame, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_window, injected.window, "f");
    }
    /**
     * Returns the status.
     *
     * @returns Status.
     */
    get status() {
        return __classPrivateFieldGet(this, _XMLHttpRequest_response, "f")?.status || 0;
    }
    /**
     * Returns the status text.
     *
     * @returns Status text.
     */
    get statusText() {
        return __classPrivateFieldGet(this, _XMLHttpRequest_response, "f")?.statusText || '';
    }
    /**
     * Returns the response.
     *
     * @returns Response.
     */
    get response() {
        if (!__classPrivateFieldGet(this, _XMLHttpRequest_response, "f")) {
            return '';
        }
        return __classPrivateFieldGet(this, _XMLHttpRequest_responseBody, "f");
    }
    /**
     * Get the response text.
     *
     * @throws {DOMException} If the response type is not text or empty.
     * @returns The response text.
     */
    get responseText() {
        if (this.responseType !== XMLHttpResponseTypeEnum_js_1.default.text && this.responseType !== '') {
            throw new DOMException_js_1.default(`Failed to read the 'responseText' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'text' (was '${this.responseType}').`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        return __classPrivateFieldGet(this, _XMLHttpRequest_responseBody, "f") ?? '';
    }
    /**
     * Get the responseXML.
     *
     * @throws {DOMException} If the response type is not text or empty.
     * @returns Response XML.
     */
    get responseXML() {
        if (this.responseType !== XMLHttpResponseTypeEnum_js_1.default.document && this.responseType !== '') {
            throw new DOMException_js_1.default(`Failed to read the 'responseXML' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'document' (was '${this.responseType}').`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        return this.responseType === '' ? null : __classPrivateFieldGet(this, _XMLHttpRequest_responseBody, "f");
    }
    /**
     * Returns the response URL.
     *
     * @returns Response URL.
     */
    get responseURL() {
        return __classPrivateFieldGet(this, _XMLHttpRequest_response, "f")?.url || '';
    }
    /**
     * Returns the ready state.
     *
     * @returns Ready state.
     */
    get readyState() {
        return __classPrivateFieldGet(this, _XMLHttpRequest_readyState, "f");
    }
    /**
     * Set response type.
     *
     * @param type Response type.
     * @throws {DOMException} If the state is not unsent or opened.
     * @throws {DOMException} If the request is synchronous.
     */
    set responseType(type) {
        // ResponseType can only be set when the state is unsent or opened.
        if (this.readyState !== XMLHttpRequestReadyStateEnum_js_1.default.opened &&
            this.readyState !== XMLHttpRequestReadyStateEnum_js_1.default.unsent) {
            throw new DOMException_js_1.default(`Failed to set the 'responseType' property on 'XMLHttpRequest': The object's state must be OPENED or UNSENT.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        // Sync requests can only have empty string or 'text' as response type.
        if (!__classPrivateFieldGet(this, _XMLHttpRequest_async, "f")) {
            throw new DOMException_js_1.default(`Failed to set the 'responseType' property on 'XMLHttpRequest': The response type cannot be changed for synchronous requests made from a document.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        __classPrivateFieldSet(this, _XMLHttpRequest_responseType, type, "f");
    }
    /**
     * Get response Type.
     *
     * @returns Response type.
     */
    get responseType() {
        return __classPrivateFieldGet(this, _XMLHttpRequest_responseType, "f");
    }
    /**
     * Opens the connection.
     *
     * @param method Connection method (eg GET, POST).
     * @param url URL for the connection.
     * @param [async=true] Asynchronous connection.
     * @param [user] Username for basic authentication (optional).
     * @param [password] Password for basic authentication (optional).
     */
    open(method, url, async = true, user, password) {
        if (!async && !!this.responseType && this.responseType !== XMLHttpResponseTypeEnum_js_1.default.text) {
            throw new DOMException_js_1.default(`Failed to execute 'open' on 'XMLHttpRequest': Synchronous requests from a document must not set a response type.`, DOMExceptionNameEnum_js_1.default.invalidAccessError);
        }
        const headers = new Headers_js_1.default();
        if (user) {
            const authBuffer = Buffer.from(`${user}:${password || ''}`);
            headers.set('Authorization', 'Basic ' + authBuffer.toString('base64'));
        }
        __classPrivateFieldSet(this, _XMLHttpRequest_async, async, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_aborted, false, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_response, null, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, null, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_abortController, new AbortController_js_1.default(), "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_request, new (__classPrivateFieldGet(this, _XMLHttpRequest_window, "f").Request)(url, {
            method,
            headers,
            signal: __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").signal,
            credentials: this.withCredentials ? 'include' : 'omit'
        }), "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.opened, "f");
    }
    /**
     * Sets a header for the request.
     *
     * @param name Header name.
     * @param value Header value.
     * @returns Header added.
     */
    setRequestHeader(name, value) {
        if (this.readyState !== XMLHttpRequestReadyStateEnum_js_1.default.opened) {
            throw new DOMException_js_1.default(`Failed to execute 'setRequestHeader' on 'XMLHttpRequest': The object's state must be OPENED.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        // TODO: Use FetchRequestHeaderUtility.removeForbiddenHeaders() instead.
        if (FetchRequestHeaderUtility_js_1.default.isHeaderForbidden(name)) {
            return false;
        }
        __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers.set(name, value);
        return true;
    }
    /**
     * Gets a header from the server response.
     *
     * @param header header Name of header to get.
     * @returns string Text of the header or null if it doesn't exist.
     */
    getResponseHeader(header) {
        return __classPrivateFieldGet(this, _XMLHttpRequest_response, "f")?.headers.get(header) ?? null;
    }
    /**
     * Gets all the response headers.
     *
     * @returns A string with all response headers separated by CR+LF.
     */
    getAllResponseHeaders() {
        if (!__classPrivateFieldGet(this, _XMLHttpRequest_response, "f")) {
            return '';
        }
        const result = [];
        for (const [name, value] of __classPrivateFieldGet(this, _XMLHttpRequest_response, "f")?.headers) {
            const lowerName = name.toLowerCase();
            if (lowerName !== 'set-cookie' && lowerName !== 'set-cookie2') {
                result.push(`${name}: ${value}`);
            }
        }
        return result.join('\r\n');
    }
    /**
     * Sends the request to the server.
     *
     * @param body Optional data to send as request body.
     */
    send(body) {
        if (this.readyState != XMLHttpRequestReadyStateEnum_js_1.default.opened) {
            throw new DOMException_js_1.default(`Failed to execute 'send' on 'XMLHttpRequest': Connection must be opened before send() is called.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        // When body is a Document, serialize it to a string.
        if (typeof body === 'object' &&
            body !== null &&
            body[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentNode) {
            body = body.documentElement.outerHTML;
        }
        if (__classPrivateFieldGet(this, _XMLHttpRequest_async, "f")) {
            __classPrivateFieldGet(this, _XMLHttpRequest_instances, "m", _XMLHttpRequest_sendAsync).call(this, body).catch((error) => {
                throw error;
            });
        }
        else {
            __classPrivateFieldGet(this, _XMLHttpRequest_instances, "m", _XMLHttpRequest_sendSync).call(this, body);
        }
    }
    /**
     * Aborts a request.
     */
    abort() {
        if (__classPrivateFieldGet(this, _XMLHttpRequest_aborted, "f")) {
            return;
        }
        __classPrivateFieldSet(this, _XMLHttpRequest_aborted, true, "f");
        __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").abort();
    }
}
_XMLHttpRequest_browserFrame = new WeakMap(), _XMLHttpRequest_window = new WeakMap(), _XMLHttpRequest_async = new WeakMap(), _XMLHttpRequest_abortController = new WeakMap(), _XMLHttpRequest_aborted = new WeakMap(), _XMLHttpRequest_request = new WeakMap(), _XMLHttpRequest_response = new WeakMap(), _XMLHttpRequest_responseType = new WeakMap(), _XMLHttpRequest_responseBody = new WeakMap(), _XMLHttpRequest_readyState = new WeakMap(), _XMLHttpRequest_instances = new WeakSet(), _XMLHttpRequest_sendAsync = 
/**
 * Sends the request to the server asynchronously.
 *
 * @param body Optional data to send as request body.
 */
async function _XMLHttpRequest_sendAsync(body) {
    const taskID = __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].startTask(() => this.abort());
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.loading, "f");
    this.dispatchEvent(new Event_js_1.default('readystatechange'));
    this.dispatchEvent(new Event_js_1.default('loadstart'));
    if (body) {
        __classPrivateFieldSet(this, _XMLHttpRequest_request, new (__classPrivateFieldGet(this, _XMLHttpRequest_window, "f").Request)(__classPrivateFieldGet(this, _XMLHttpRequest_request, "f").url, {
            method: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").method,
            headers: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers,
            signal: __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").signal,
            credentials: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").credentials,
            body
        }), "f");
    }
    __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").signal.addEventListener('abort', () => {
        __classPrivateFieldSet(this, _XMLHttpRequest_aborted, true, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.unsent, "f");
        this.dispatchEvent(new Event_js_1.default('abort'));
        this.dispatchEvent(new Event_js_1.default('loadend'));
        this.dispatchEvent(new Event_js_1.default('readystatechange'));
        __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
    });
    const onError = (error) => {
        if (error instanceof DOMException_js_1.default && error.name === DOMExceptionNameEnum_js_1.default.abortError) {
            if (__classPrivateFieldGet(this, _XMLHttpRequest_aborted, "f")) {
                return;
            }
            __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.unsent, "f");
            this.dispatchEvent(new Event_js_1.default('abort'));
        }
        else {
            __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.done, "f");
            this.dispatchEvent(new ErrorEvent_js_1.default('error', { error, message: error.message }));
        }
        this.dispatchEvent(new Event_js_1.default('loadend'));
        this.dispatchEvent(new Event_js_1.default('readystatechange'));
        __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
    };
    const fetch = new Fetch_js_1.default({
        browserFrame: __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f"),
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        url: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").url,
        init: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f")
    });
    try {
        __classPrivateFieldSet(this, _XMLHttpRequest_response, await fetch.send(), "f");
    }
    catch (error) {
        onError(error);
        return;
    }
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.headersRecieved, "f");
    this.dispatchEvent(new Event_js_1.default('readystatechange'));
    const contentLength = __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").headers.get('Content-Length');
    const contentLengthNumber = contentLength !== null && !isNaN(Number(contentLength)) ? Number(contentLength) : null;
    let loaded = 0;
    let data = Buffer.from([]);
    if (__classPrivateFieldGet(this, _XMLHttpRequest_response, "f").body) {
        let eventError;
        try {
            for await (const chunk of __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").body) {
                data = Buffer.concat([data, typeof chunk === 'string' ? Buffer.from(chunk) : chunk]);
                loaded += chunk.length;
                // We need to re-throw the error as we don't want it to be caught by the try/catch.
                try {
                    this.dispatchEvent(new ProgressEvent_js_1.default('progress', {
                        lengthComputable: contentLengthNumber !== null,
                        loaded,
                        total: contentLengthNumber !== null ? contentLengthNumber : 0
                    }));
                }
                catch (error) {
                    eventError = error;
                    throw error;
                }
            }
        }
        catch (error) {
            if (error === eventError) {
                throw error;
            }
            onError(error);
            return;
        }
    }
    __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, XMLHttpRequestResponseDataParser_js_1.default.parse({
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        responseType: __classPrivateFieldGet(this, _XMLHttpRequest_responseType, "f"),
        data,
        contentType: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").headers.get('Content-Type') || __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers.get('Content-Type')
    }), "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.done, "f");
    this.dispatchEvent(new Event_js_1.default('readystatechange'));
    this.dispatchEvent(new Event_js_1.default('load'));
    this.dispatchEvent(new Event_js_1.default('loadend'));
    __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
}, _XMLHttpRequest_sendSync = function _XMLHttpRequest_sendSync(body) {
    if (body) {
        __classPrivateFieldSet(this, _XMLHttpRequest_request, new (__classPrivateFieldGet(this, _XMLHttpRequest_window, "f").Request)(__classPrivateFieldGet(this, _XMLHttpRequest_request, "f").url, {
            method: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").method,
            headers: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers,
            signal: __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").signal,
            credentials: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").credentials,
            body
        }), "f");
    }
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.loading, "f");
    const fetch = new SyncFetch_js_1.default({
        browserFrame: __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f"),
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        url: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").url,
        init: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f")
    });
    try {
        __classPrivateFieldSet(this, _XMLHttpRequest_response, fetch.send(), "f");
    }
    catch (error) {
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.done, "f");
        this.dispatchEvent(new ErrorEvent_js_1.default('error', { error, message: error.message }));
        this.dispatchEvent(new Event_js_1.default('loadend'));
        this.dispatchEvent(new Event_js_1.default('readystatechange'));
        return;
    }
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.headersRecieved, "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, XMLHttpRequestResponseDataParser_js_1.default.parse({
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        responseType: __classPrivateFieldGet(this, _XMLHttpRequest_responseType, "f"),
        data: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").body,
        contentType: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").headers.get('Content-Type') || __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers.get('Content-Type')
    }), "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum_js_1.default.done, "f");
    this.dispatchEvent(new Event_js_1.default('readystatechange'));
    this.dispatchEvent(new Event_js_1.default('load'));
    this.dispatchEvent(new Event_js_1.default('loadend'));
};
// Constants
XMLHttpRequest.UNSENT = XMLHttpRequestReadyStateEnum_js_1.default.unsent;
XMLHttpRequest.OPENED = XMLHttpRequestReadyStateEnum_js_1.default.opened;
XMLHttpRequest.HEADERS_RECEIVED = XMLHttpRequestReadyStateEnum_js_1.default.headersRecieved;
XMLHttpRequest.LOADING = XMLHttpRequestReadyStateEnum_js_1.default.loading;
XMLHttpRequest.DONE = XMLHttpRequestReadyStateEnum_js_1.default.done;
exports.default = XMLHttpRequest;
//# sourceMappingURL=XMLHttpRequest.cjs.map