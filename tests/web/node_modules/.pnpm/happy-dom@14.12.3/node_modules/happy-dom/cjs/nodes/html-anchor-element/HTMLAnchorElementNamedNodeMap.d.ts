import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.cjs';
import HTMLAnchorElement from './HTMLAnchorElement.cjs';
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