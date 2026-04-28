/// <reference types="node" />
import AsyncTaskManager from '../../async-task-manager/AsyncTaskManager.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import Document from '../../nodes/document/Document.cjs';
import IBrowserPage from './IBrowserPage.cjs';
import Response from '../../fetch/Response.cjs';
import IGoToOptions from './IGoToOptions.cjs';
import { Script } from 'vm';
import IReloadOptions from './IReloadOptions.cjs';
import BrowserFrameExceptionObserver from '../utilities/BrowserFrameExceptionObserver.cjs';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.cjs';
/**
 * Browser frame.
 */
export default interface IBrowserFrame {
    readonly childFrames: IBrowserFrame[];
    readonly parentFrame: IBrowserFrame | null;
    readonly page: IBrowserPage;
    readonly window: BrowserWindow;
    readonly document: Document;
    content: string;
    url: string;
    [PropertySymbol.asyncTaskManager]: AsyncTaskManager;
    [PropertySymbol.exceptionObserver]: BrowserFrameExceptionObserver | null;
    [PropertySymbol.listeners]: {
        navigation: Array<() => void>;
    };
    [PropertySymbol.openerFrame]: IBrowserFrame | null;
    [PropertySymbol.openerWindow]: BrowserWindow | CrossOriginBrowserWindow | null;
    [PropertySymbol.popup]: boolean;
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
     */
    goto(url: string, options?: IGoToOptions): Promise<Response | null>;
    /**
     * Reloads the current frame.
     *
     * @param [options] Options.
     */
    reload(options: IReloadOptions): Promise<Response | null>;
}
//# sourceMappingURL=IBrowserFrame.d.ts.map