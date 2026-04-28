import IBrowserFrame from '../../browser/types/IBrowserFrame.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import Headers from '../Headers.cjs';
import Request from '../Request.cjs';
/**
 * Fetch request header utility.
 */
export default class FetchRequestHeaderUtility {
    /**
     * Validates request headers.
     *
     * @param headers Headers.
     */
    static removeForbiddenHeaders(headers: Headers): void;
    /**
     * Returns "true" if the header is forbidden.
     *
     * @param name Header name.
     * @returns "true" if the header is forbidden.
     */
    static isHeaderForbidden(name: string): boolean;
    /**
     * Returns request headers.
     *
     * @param options Options.
     * @param options.browserFrame Browser frame.
     * @param options.window Window.
     * @param options.request Request.
     * @returns Headers.
     */
    static getRequestHeaders(options: {
        browserFrame: IBrowserFrame;
        window: BrowserWindow;
        request: Request;
    }): {
        [key: string]: string;
    };
}
//# sourceMappingURL=FetchRequestHeaderUtility.d.ts.map