var _a, _b, _c;
import DocumentFragment from '../document-fragment/DocumentFragment.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import XMLParser from '../../xml-parser/XMLParser.js';
import XMLSerializer from '../../xml-serializer/XMLSerializer.js';
/**
 * ShadowRoot.
 */
export default class ShadowRoot extends DocumentFragment {
    constructor() {
        super(...arguments);
        // Events
        this.onslotchange = null;
        // Internal properties
        this[_a] = [];
        this[_b] = 'open';
        this[_c] = null;
    }
    /**
     * Returns mode.
     *
     * @returns Mode.
     */
    get mode() {
        return this[PropertySymbol.mode];
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        return this[PropertySymbol.host];
    }
    /**
     * Returns inner HTML.
     *
     * @returns HTML.
     */
    get innerHTML() {
        const xmlSerializer = new XMLSerializer({
            escapeEntities: false
        });
        let xml = '';
        for (const node of this[PropertySymbol.childNodes]) {
            xml += xmlSerializer.serializeToString(node);
        }
        return xml;
    }
    /**
     * Sets inner HTML.
     *
     * @param html HTML.
     */
    set innerHTML(html) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        XMLParser.parse(this[PropertySymbol.ownerDocument], html, { rootNode: this });
    }
    /**
     * Returns adopted style sheets.
     *
     * @returns Adopted style sheets.
     */
    get adoptedStyleSheets() {
        return this[PropertySymbol.adoptedStyleSheets];
    }
    /**
     * Sets adopted style sheets.
     *
     * @param value Adopted style sheets.
     */
    set adoptedStyleSheets(value) {
        this[PropertySymbol.adoptedStyleSheets] = value;
    }
    /**
     * Returns active element.
     *
     * @returns Active element.
     */
    get activeElement() {
        const activeElement = this[PropertySymbol.ownerDocument][PropertySymbol.activeElement];
        if (activeElement &&
            activeElement[PropertySymbol.isConnected] &&
            activeElement.getRootNode() === this) {
            return activeElement;
        }
        return null;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return this.innerHTML;
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.adoptedStyleSheets, _b = PropertySymbol.mode, _c = PropertySymbol.host, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.mode] = this.mode;
        return clone;
    }
}
//# sourceMappingURL=ShadowRoot.js.map