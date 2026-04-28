import HTMLLinkElement from './HTMLLinkElement.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
/**
 * Helper class for getting the URL relative to a Location object.
 */
export default class HTMLLinkElementStyleSheetLoader {
    #private;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Element.
     * @param options.browserFrame Browser frame.
     */
    constructor(options: {
        element: HTMLLinkElement;
        browserFrame: IBrowserFrame;
    });
    /**
     * Returns a URL relative to the given Location object.
     *
     * @param url URL.
     * @param rel Rel.
     */
    loadStyleSheet(url: string | null, rel: string | null): Promise<void>;
}
//# sourceMappingURL=HTMLLinkElementStyleSheetLoader.d.ts.map