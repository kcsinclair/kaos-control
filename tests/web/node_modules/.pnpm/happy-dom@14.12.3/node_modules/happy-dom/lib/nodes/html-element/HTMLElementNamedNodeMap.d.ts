import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ElementNamedNodeMap from '../element/ElementNamedNodeMap.js';
import HTMLElement from './HTMLElement.js';
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