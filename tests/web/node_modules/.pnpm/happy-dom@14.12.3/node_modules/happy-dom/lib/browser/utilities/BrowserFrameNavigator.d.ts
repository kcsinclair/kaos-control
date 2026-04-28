import IBrowserFrame from '../types/IBrowserFrame.js';
import IGoToOptions from '../types/IGoToOptions.js';
import Response from '../../fetch/Response.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import FormData from '../../form-data/FormData.js';
/**
 * Browser frame navigation utility.
 */
export default class BrowserFrameNavigator {
    /**
     * Navigates to a page.
     *
     * @throws Error if the request can't be resolved (because of SSL error or similar). It will not throw if the response is not ok.
     * @param options Options.
     * @param options.windowClass Window class.
     * @param options.frame Frame.
     * @param options.url URL.
     * @param [options.formData] Form data.
     * @param [options.method] Method.
     * @param [options.goToOptions] Go to options.
     * @returns Response.
     */
    static navigate(options: {
        windowClass: new (browserFrame: IBrowserFrame, options?: {
            url?: string;
            width?: number;
            height?: number;
        }) => BrowserWindow;
        frame: IBrowserFrame;
        url: string;
        goToOptions?: IGoToOptions;
        method?: string;
        formData?: FormData;
    }): Promise<Response | null>;
}
//# sourceMappingURL=BrowserFrameNavigator.d.ts.map