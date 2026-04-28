import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.cjs';
import HTMLScriptElement from './HTMLScriptElement.cjs';
import HTMLScriptElementScriptLoader from './HTMLScriptElementScriptLoader.cjs';
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