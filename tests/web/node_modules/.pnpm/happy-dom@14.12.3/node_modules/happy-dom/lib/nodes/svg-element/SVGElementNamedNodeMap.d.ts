import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ElementNamedNodeMap from '../element/ElementNamedNodeMap.js';
import SVGElement from './SVGElement.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class SVGElementNamedNodeMap extends ElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: SVGElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=SVGElementNamedNodeMap.d.ts.map