/// <reference types="node" />
/// <reference types="node" />
import * as PropertySymbol from '../PropertySymbol.cjs';
import Document from '../nodes/document/Document.cjs';
import IRequestInit from './types/IRequestInit.cjs';
import URL from '../url/URL.cjs';
import IRequestInfo from './types/IRequestInfo.cjs';
import Headers from './Headers.cjs';
import AbortSignal from './AbortSignal.cjs';
import { ReadableStream } from 'stream/web';
import Blob from '../file/Blob.cjs';
import IRequestReferrerPolicy from './types/IRequestReferrerPolicy.cjs';
import IRequestRedirect from './types/IRequestRedirect.cjs';
import IRequestCredentials from './types/IRequestCredentials.cjs';
import FormData from '../form-data/FormData.cjs';
import AsyncTaskManager from '../async-task-manager/AsyncTaskManager.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * Fetch request.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/request.js
 *
 * @see https://fetch.spec.whatwg.org/#request-class
 */
export default class Request implements Request {
    #private;
    readonly method: string;
    readonly body: ReadableStream | null;
    readonly headers: Headers;
    readonly redirect: IRequestRedirect;
    readonly referrerPolicy: IRequestReferrerPolicy;
    readonly signal: AbortSignal;
    readonly bodyUsed: boolean;
    readonly credentials: IRequestCredentials;
    [PropertySymbol.contentLength]: number | null;
    [PropertySymbol.contentType]: string | null;
    [PropertySymbol.referrer]: '' | 'no-referrer' | 'client' | URL;
    [PropertySymbol.url]: URL;
    [PropertySymbol.bodyBuffer]: Buffer | null;
    /**
     * Constructor.
     *
     * @param injected Injected properties.
     * @param injected.window
     * @param input Input.
     * @param injected.asyncTaskManager
     * @param [init] Init.
     */
    constructor(injected: {
        window: BrowserWindow;
        asyncTaskManager: AsyncTaskManager;
    }, input: IRequestInfo, init?: IRequestInit);
    /**
     * Returns owner document.
     */
    protected get [PropertySymbol.ownerDocument](): Document;
    /**
     * Returns referrer.
     *
     * @returns Referrer.
     */
    get referrer(): string;
    /**
     * Returns URL.
     *
     * @returns URL.
     */
    get url(): string;
    /**
     * Returns string tag.
     *
     * @returns String tag.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Returns array buffer.
     *
     * @returns Array buffer.
     */
    arrayBuffer(): Promise<ArrayBuffer>;
    /**
     * Returns blob.
     *
     * @returns Blob.
     */
    blob(): Promise<Blob>;
    /**
     * Returns buffer.
     *
     * @returns Buffer.
     */
    buffer(): Promise<Buffer>;
    /**
     * Returns text.
     *
     * @returns Text.
     */
    text(): Promise<string>;
    /**
     * Returns json.
     *
     * @returns JSON.
     */
    json(): Promise<string>;
    /**
     * Returns FormData.
     *
     * @returns FormData.
     */
    formData(): Promise<FormData>;
    /**
     * Clones request.
     *
     * @returns Clone.
     */
    clone(): Request;
}
//# sourceMappingURL=Request.d.ts.map