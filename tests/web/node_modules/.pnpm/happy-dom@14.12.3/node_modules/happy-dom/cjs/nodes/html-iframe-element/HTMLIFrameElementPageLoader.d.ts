import BrowserWindow from '../../window/BrowserWindow.cjs';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.cjs';
import IBrowserFrame from '../../browser/types/IBrowserFrame.cjs';
import HTMLIFrameElement from './HTMLIFrameElement.cjs';
/**
 * HTML Iframe page loader.
 */
export default class HTMLIFrameElementPageLoader {
    #private;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Iframe element.
     * @param options.browserParentFrame Main browser frame.
     * @param options.contentWindowContainer Content window container.
     * @param options.contentWindowContainer.window Content window.
     */
    constructor(options: {
        element: HTMLIFrameElement;
        browserParentFrame: IBrowserFrame;
        contentWindowContainer: {
            window: BrowserWindow | CrossOriginBrowserWindow | null;
        };
    });
    /**
     * Loads an iframe page.
     */
    loadPage(): void;
    /**
     * Unloads an iframe page.
     */
    unloadPage(): void;
}
//# sourceMappingURL=HTMLIFrameElementPageLoader.d.ts.map