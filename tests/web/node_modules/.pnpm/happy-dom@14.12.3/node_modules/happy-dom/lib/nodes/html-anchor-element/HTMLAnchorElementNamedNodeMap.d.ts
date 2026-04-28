import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
import HTMLAnchorElement from './HTMLAnchorElement.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLAnchorElementNamedNodeMap extends HTMLElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: HTMLAnchorElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLAnchorElementNamedNodeMap.d.ts.map