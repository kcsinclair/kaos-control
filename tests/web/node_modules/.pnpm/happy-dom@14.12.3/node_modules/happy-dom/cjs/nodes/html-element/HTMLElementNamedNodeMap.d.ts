import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import ElementNamedNodeMap from '../element/ElementNamedNodeMap.cjs';
import HTMLElement from './HTMLElement.cjs';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLElementNamedNodeMap extends ElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: HTMLElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLElementNamedNodeMap.d.ts.map