import HTMLScriptElement from './HTMLScriptElement.js';
import IBrowserFrame from '../../browser/types/IBrowserFrame.js';
/**
 * Helper class for getting the URL relative to a Location object.
 */
export default class HTMLScriptElementScriptLoader {
    #private;
    /**
     * Constructor.
     *
     * @param options Options.
     * @param options.element Element.
     * @param options.browserFrame Browser frame.
     */
    constructor(options: {
        element: HTMLScriptElement;
        browserFrame: IBrowserFrame;
    });
    /**
     * Returns a URL relative to the given Location object.
     *
     * @param url URL.
     */
    loadScript(url: string): Promise<void>;
}
//# sourceMappingURL=HTMLScriptElementScriptLoader.d.ts.map