import Attr from '../attr/Attr.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import ElementNamedNodeMap from '../element/ElementNamedNodeMap.cjs';
import SVGElement from './SVGElement.cjs';
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