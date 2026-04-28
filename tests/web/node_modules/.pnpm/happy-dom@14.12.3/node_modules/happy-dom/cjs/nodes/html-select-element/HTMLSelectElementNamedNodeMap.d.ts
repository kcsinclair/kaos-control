import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.cjs';
import HTMLSelectElement from './HTMLSelectElement.cjs';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLSelectElementNamedNodeMap extends HTMLElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: HTMLSelectElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLSelectElementNamedNodeMap.d.ts.map