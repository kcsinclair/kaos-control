import * as PropertySymbol from '../../PropertySymbol.js';
import NamedNodeMap from '../../named-node-map/NamedNodeMap.js';
import Attr from '../attr/Attr.js';
import Element from './Element.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class ElementNamedNodeMap extends NamedNodeMap {
    protected [PropertySymbol.ownerElement]: Element;
    /**
     * Constructor.
     *
     * @param ownerElement Owner element.
     */
    constructor(ownerElement: Element);
    /**
     * @override
     */
    getNamedItem(name: string): Attr | null;
    /**
     * @override
     */
    getNamedItemNS(namespace: string, localName: string): Attr | null;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
    /**
     * @override
     */
    removeNamedItemNS(namespace: string, localName: string): Attr | null;
    /**
     * Returns attribute name.
     *
     * @param name Name.
     * @returns Attribute name based on namespace.
     */
    protected [PropertySymbol.getAttributeName](name: any): string;
}
//# sourceMappingURL=ElementNamedNodeMap.d.ts.map