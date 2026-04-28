import IRequestInit from './types/IRequestInit.cjs';
import IRequestInfo from './types/IRequestInfo.cjs';
import IBrowserFrame from '../browser/types/IBrowserFrame.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
import ISyncResponse from './types/ISyncResponse.cjs';
/**
 * Handles synchrounous fetch requests.
 */
export default class SyncFetch {
    #private;
    private request;
    private redirectCount;
    private disableCache;
    private disableCrossOriginPolicy;
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
    constructor(options: {
        browserFrame: IBrowserFrame;
        window: BrowserWindow;
        url: IRequestInfo;
        init?: IRequestInit;
        redirectCount?: number;
        contentType?: string;
        disableCache?: boolean;
        disableCrossOriginPolicy?: boolean;
    });
    /**
     * Sends request.
     *
     * @returns Response.
     */
    send(): ISyncResponse;
    /**
     * Returns cached response.
     *
     * @returns Response.
     */
    getCachedResponse(): ISyncResponse | null;
    /**
     * Checks if the request complies with the Cross-Origin policy.
     *
     * @returns True if it complies with the policy.
     */
    private compliesWithCrossOriginPolicy;
    /**
     * Sends request.
     *
     * @returns Response.
     */
    sendRequest(): ISyncResponse;
    /**
     * Parses response body.
     *
     * @param options Options.
     * @param options.headers Headers.
     * @param options.status Status.
     * @param options.body Body.
     * @returns Parsed body.
     */
    private parseIResponseBody;
    /**
     * Handles redirect response.
     *
     * @param response Response.
     * @returns Redirected response or null.
     */
    private handleRedirectResponse;
}
//# sourceMappingURL=SyncFetch.d.ts.map