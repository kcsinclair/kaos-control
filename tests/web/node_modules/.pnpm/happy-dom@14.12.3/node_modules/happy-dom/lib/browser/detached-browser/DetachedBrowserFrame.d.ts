/// <reference types="node" resolution-mode="require"/>
import DetachedBrowserPage from './DetachedBrowserPage.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import AsyncTaskManager from '../../async-task-manager/AsyncTaskManager.js';
import IBrowserFrame from '../types/IBrowserFrame.js';
import Response from '../../fetch/Response.js';
import IGoToOptions from '../types/IGoToOptions.js';
import { Script } from 'vm';
import BrowserWindow from '../../window/BrowserWindow.js';
import IReloadOptions from '../types/IReloadOptions.js';
import BrowserFrameExceptionObserver from '../utilities/BrowserFrameExceptionObserver.js';
import Document from '../../nodes/document/Document.js';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.js';
/**
 * Browser frame used when constructing a Window instance without a browser.
 */
export default class DetachedBrowserFrame implements IBrowserFrame {
    readonly childFrames: DetachedBrowserFrame[];
    readonly parentFrame: DetachedBrowserFrame | null;
    readonly page: DetachedBrowserPage;
    window: BrowserWindow;
    [PropertySymbol.asyncTaskManager]: AsyncTaskManager;
    [PropertySymbol.exceptionObserver]: BrowserFrameExceptionObserver | null;
    [PropertySymbol.listeners]: {
        navigation: Array<() => void>;
    };
    [PropertySymbol.openerFrame]: IBrowserFrame | null;
    [PropertySymbol.openerWindow]: BrowserWindow | CrossOriginBrowserWindow | null;
    [PropertySymbol.popup]: boolean;
    /**
     * Constructor.
     *
     * @param page Page.
     * @param [window] Window.
     */
    constructor(page: DetachedBrowserPage);
    /**
     * Returns the content.
     *
     * @returns Content.
     */
    get content(): string;
    /**
     * Sets the content.
     *
     * @param content Content.
     */
    set content(content: string);
    /**
     * Returns the URL.
     *
     * @returns URL.
     */
    get url(): string;
    /**
     * Sets the content.
     *
     * @param url URL.
     */
    set url(url: string);
    /**
     * Returns document.
     *
     * @returns Document.
     */
    get document(): Document;
    /**
     * Returns a promise that is resolved when all resources has been loaded, fetch has completed, and all async tasks such as timers are complete.
     */
    waitUntilComplete(): Promise<void>;
    /**
     * Returns a promise that is resolved when the frame has navigated and the response HTML has been written to the document.
     */
    waitForNavigation(): Promise<void>;
    /**
     * Aborts all ongoing operations.
     */
    abort(): Promise<void>;
    /**
     * Evaluates code or a VM Script in the page's context.
     *
     * @param script Script.
     * @returns Result.
     */
    evaluate(script: string | Script): any;
    /**
     * Go to a page.
     *
     * @param url URL.
     * @param [options] Options.
     * @returns Response.
     */
    goto(url: string, options?: IGoToOptions): Promise<Response | null>;
    /**
     * Reloads the current frame.
     *
     * @param [options] Options.
     * @returns Response.
     */
    reload(options: IReloadOptions): Promise<Response | null>;
}
//# sourceMappingURL=DetachedBrowserFrame.d.ts.map