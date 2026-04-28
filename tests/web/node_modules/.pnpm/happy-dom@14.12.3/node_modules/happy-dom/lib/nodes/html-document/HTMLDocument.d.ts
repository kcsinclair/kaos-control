import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Document from '../document/Document.js';
/**
 * Document.
 */
export default class HTMLDocument extends Document {
    /**
     * Constructor.
     *
     * @param injected Injected properties.
     * @param injected.browserFrame Browser frame.
     * @param injected.window Window.
     */
    constructor(injected: {
        browserFrame: IBrowserFrame;
        window: BrowserWindow;
    });
}
//# sourceMappingURL=HTMLDocument.d.ts.map