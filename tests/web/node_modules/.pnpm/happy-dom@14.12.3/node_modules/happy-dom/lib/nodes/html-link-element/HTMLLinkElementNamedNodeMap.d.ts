import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
import HTMLLinkElement from './HTMLLinkElement.js';
import HTMLLinkElementStyleSheetLoader from './HTMLLinkElementStyleSheetLoader.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLLinkElementNamedNodeMap extends HTMLElementNamedNodeMap {
    #private;
    protected [PropertySymbol.ownerElement]: HTMLLinkElement;
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param stylesheetLoader Stylesheet loader.
     * @param styleSheetLoader
     */
    constructor(ownerElement: HTMLLinkElement, styleSheetLoader: HTMLLinkElementStyleSheetLoader);
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLLinkElementNamedNodeMap.d.ts.map