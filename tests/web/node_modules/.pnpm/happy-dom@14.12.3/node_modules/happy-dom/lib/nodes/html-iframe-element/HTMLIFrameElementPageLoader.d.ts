import BrowserWindow from '../../window/BrowserWindow.js';
import CrossOriginBrowserWindow from '../../window/CrossOriginBrowserWindow.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
import HTMLIFrameElement from './HTMLIFrameElement.js';
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