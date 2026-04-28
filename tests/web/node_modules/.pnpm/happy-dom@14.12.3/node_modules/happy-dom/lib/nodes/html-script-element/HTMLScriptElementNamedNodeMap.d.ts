import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
import HTMLScriptElement from './HTMLScriptElement.js';
import HTMLScriptElementScriptLoader from './HTMLScriptElementScriptLoader.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLScriptElementNamedNodeMap extends HTMLElementNamedNodeMap {
    #private;
    protected [PropertySymbol.ownerElement]: HTMLScriptElement;
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     * @param scriptLoader Script loader.
     */
    constructor(ownerElement: HTMLScriptElement, scriptLoader: HTMLScriptElementScriptLoader);
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
}
//# sourceMappingURL=HTMLScriptElementNamedNodeMap.d.ts.map