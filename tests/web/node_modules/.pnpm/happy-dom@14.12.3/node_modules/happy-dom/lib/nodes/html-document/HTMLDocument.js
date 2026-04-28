import Document from '../document/Document.js';
import * as PropertySymbol from '../../PropertySymbol.js';
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
    constructor(injected) {
        super(injected);
        // Default document elements
        const doctype = this[PropertySymbol.implementation].createDocumentType('html', '', '');
        const documentElement = this.createElement('html');
        const bodyElement = this.createElement('body');
        const headElement = this.createElement('head');
        this.appendChild(doctype);
        this.appendChild(documentElement);
        documentElement.appendChild(headElement);
        documentElement.appendChild(bodyElement);
    }
}
//# sourceMappingURL=HTMLDocument.js.map