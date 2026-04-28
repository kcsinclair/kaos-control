import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.cjs';
import HTMLOptionElement from './HTMLOptionElement.cjs';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLOptionElementNamedNodeMap extends HTMLElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: HTMLOptionElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLOptionElementNamedNodeMap.d.ts.map