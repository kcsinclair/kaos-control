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
var _XMLHttpRequest_instances, _XMLHttpRequest_browserFrame, _XMLHttpRequest_window, _XMLHttpRequest_async, _XMLHttpRequest_abortController, _XMLHttpRequest_aborted, _XMLHttpRequest_request, _XMLHttpRequest_response, _XMLHttpRequest_responseType, _XMLHttpRequest_responseBody, _XMLHttpRequest_readyState, _XMLHttpRequest_sendAsync, _XMLHttpRequest_sendSync;
import XMLHttpRequestEventTarget from './XMLHttpRequestEventTarget.js';
import * as PropertySymbol from '../PropertySymbol.js';
import XMLHttpRequestReadyStateEnum from './XMLHttpRequestReadyStateEnum.js';
import Event from '../event/Event.js';
import XMLHttpRequestUpload from './XMLHttpRequestUpload.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import XMLHttpResponseTypeEnum from './XMLHttpResponseTypeEnum.js';
import ErrorEvent from '../event/events/ErrorEvent.js';
import Headers from '../fetch/Headers.js';
import Fetch from '../fetch/Fetch.js';
import SyncFetch from '../fetch/SyncFetch.js';
import AbortController from '../fetch/AbortController.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import XMLHttpRequestResponseDataParser from './XMLHttpRequestResponseDataParser.js';
import FetchRequestHeaderUtility from '../fetch/utilities/FetchRequestHeaderUtility.js';
/**
 * XMLHttpRequest.
 *
 * Based on:
 * https://github.com/mjwwit/node-XMLHttpRequest/blob/master/lib/XMLHttpRequest.js
 */
class XMLHttpRequest extends XMLHttpRequestEventTarget {
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
        this.upload = new XMLHttpRequestUpload();
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
        _XMLHttpRequest_readyState.set(this, XMLHttpRequestReadyStateEnum.unsent);
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
        if (this.responseType !== XMLHttpResponseTypeEnum.text && this.responseType !== '') {
            throw new DOMException(`Failed to read the 'responseText' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'text' (was '${this.responseType}').`, DOMExceptionNameEnum.invalidStateError);
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
        if (this.responseType !== XMLHttpResponseTypeEnum.document && this.responseType !== '') {
            throw new DOMException(`Failed to read the 'responseXML' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'document' (was '${this.responseType}').`, DOMExceptionNameEnum.invalidStateError);
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
        if (this.readyState !== XMLHttpRequestReadyStateEnum.opened &&
            this.readyState !== XMLHttpRequestReadyStateEnum.unsent) {
            throw new DOMException(`Failed to set the 'responseType' property on 'XMLHttpRequest': The object's state must be OPENED or UNSENT.`, DOMExceptionNameEnum.invalidStateError);
        }
        // Sync requests can only have empty string or 'text' as response type.
        if (!__classPrivateFieldGet(this, _XMLHttpRequest_async, "f")) {
            throw new DOMException(`Failed to set the 'responseType' property on 'XMLHttpRequest': The response type cannot be changed for synchronous requests made from a document.`, DOMExceptionNameEnum.invalidStateError);
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
        if (!async && !!this.responseType && this.responseType !== XMLHttpResponseTypeEnum.text) {
            throw new DOMException(`Failed to execute 'open' on 'XMLHttpRequest': Synchronous requests from a document must not set a response type.`, DOMExceptionNameEnum.invalidAccessError);
        }
        const headers = new Headers();
        if (user) {
            const authBuffer = Buffer.from(`${user}:${password || ''}`);
            headers.set('Authorization', 'Basic ' + authBuffer.toString('base64'));
        }
        __classPrivateFieldSet(this, _XMLHttpRequest_async, async, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_aborted, false, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_response, null, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, null, "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_abortController, new AbortController(), "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_request, new (__classPrivateFieldGet(this, _XMLHttpRequest_window, "f").Request)(url, {
            method,
            headers,
            signal: __classPrivateFieldGet(this, _XMLHttpRequest_abortController, "f").signal,
            credentials: this.withCredentials ? 'include' : 'omit'
        }), "f");
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.opened, "f");
    }
    /**
     * Sets a header for the request.
     *
     * @param name Header name.
     * @param value Header value.
     * @returns Header added.
     */
    setRequestHeader(name, value) {
        if (this.readyState !== XMLHttpRequestReadyStateEnum.opened) {
            throw new DOMException(`Failed to execute 'setRequestHeader' on 'XMLHttpRequest': The object's state must be OPENED.`, DOMExceptionNameEnum.invalidStateError);
        }
        // TODO: Use FetchRequestHeaderUtility.removeForbiddenHeaders() instead.
        if (FetchRequestHeaderUtility.isHeaderForbidden(name)) {
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
        if (this.readyState != XMLHttpRequestReadyStateEnum.opened) {
            throw new DOMException(`Failed to execute 'send' on 'XMLHttpRequest': Connection must be opened before send() is called.`, DOMExceptionNameEnum.invalidStateError);
        }
        // When body is a Document, serialize it to a string.
        if (typeof body === 'object' &&
            body !== null &&
            body[PropertySymbol.nodeType] === NodeTypeEnum.documentNode) {
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
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.loading, "f");
    this.dispatchEvent(new Event('readystatechange'));
    this.dispatchEvent(new Event('loadstart'));
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
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.unsent, "f");
        this.dispatchEvent(new Event('abort'));
        this.dispatchEvent(new Event('loadend'));
        this.dispatchEvent(new Event('readystatechange'));
        __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
    });
    const onError = (error) => {
        if (error instanceof DOMException && error.name === DOMExceptionNameEnum.abortError) {
            if (__classPrivateFieldGet(this, _XMLHttpRequest_aborted, "f")) {
                return;
            }
            __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.unsent, "f");
            this.dispatchEvent(new Event('abort'));
        }
        else {
            __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.done, "f");
            this.dispatchEvent(new ErrorEvent('error', { error, message: error.message }));
        }
        this.dispatchEvent(new Event('loadend'));
        this.dispatchEvent(new Event('readystatechange'));
        __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f")[PropertySymbol.asyncTaskManager].endTask(taskID);
    };
    const fetch = new Fetch({
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
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.headersRecieved, "f");
    this.dispatchEvent(new Event('readystatechange'));
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
                    this.dispatchEvent(new ProgressEvent('progress', {
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
    __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, XMLHttpRequestResponseDataParser.parse({
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        responseType: __classPrivateFieldGet(this, _XMLHttpRequest_responseType, "f"),
        data,
        contentType: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").headers.get('Content-Type') || __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers.get('Content-Type')
    }), "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.done, "f");
    this.dispatchEvent(new Event('readystatechange'));
    this.dispatchEvent(new Event('load'));
    this.dispatchEvent(new Event('loadend'));
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
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.loading, "f");
    const fetch = new SyncFetch({
        browserFrame: __classPrivateFieldGet(this, _XMLHttpRequest_browserFrame, "f"),
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        url: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").url,
        init: __classPrivateFieldGet(this, _XMLHttpRequest_request, "f")
    });
    try {
        __classPrivateFieldSet(this, _XMLHttpRequest_response, fetch.send(), "f");
    }
    catch (error) {
        __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.done, "f");
        this.dispatchEvent(new ErrorEvent('error', { error, message: error.message }));
        this.dispatchEvent(new Event('loadend'));
        this.dispatchEvent(new Event('readystatechange'));
        return;
    }
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.headersRecieved, "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_responseBody, XMLHttpRequestResponseDataParser.parse({
        window: __classPrivateFieldGet(this, _XMLHttpRequest_window, "f"),
        responseType: __classPrivateFieldGet(this, _XMLHttpRequest_responseType, "f"),
        data: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").body,
        contentType: __classPrivateFieldGet(this, _XMLHttpRequest_response, "f").headers.get('Content-Type') || __classPrivateFieldGet(this, _XMLHttpRequest_request, "f").headers.get('Content-Type')
    }), "f");
    __classPrivateFieldSet(this, _XMLHttpRequest_readyState, XMLHttpRequestReadyStateEnum.done, "f");
    this.dispatchEvent(new Event('readystatechange'));
    this.dispatchEvent(new Event('load'));
    this.dispatchEvent(new Event('loadend'));
};
// Constants
XMLHttpRequest.UNSENT = XMLHttpRequestReadyStateEnum.unsent;
XMLHttpRequest.OPENED = XMLHttpRequestReadyStateEnum.opened;
XMLHttpRequest.HEADERS_RECEIVED = XMLHttpRequestReadyStateEnum.headersRecieved;
XMLHttpRequest.LOADING = XMLHttpRequestReadyStateEnum.loading;
XMLHttpRequest.DONE = XMLHttpRequestReadyStateEnum.done;
export default XMLHttpRequest;
//# sourceMappingURL=XMLHttpRequest.js.map