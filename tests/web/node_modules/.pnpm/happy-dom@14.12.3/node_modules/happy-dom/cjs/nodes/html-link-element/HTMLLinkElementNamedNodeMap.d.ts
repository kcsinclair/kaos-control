import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.cjs';
import HTMLLinkElement from './HTMLLinkElement.cjs';
import HTMLLinkElementStyleSheetLoader from './HTMLLinkElementStyleSheetLoader.cjs';
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