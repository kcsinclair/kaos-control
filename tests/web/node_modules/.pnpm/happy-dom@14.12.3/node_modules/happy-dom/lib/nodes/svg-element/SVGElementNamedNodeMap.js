import * as PropertySymbol from '../../PropertySymbol.js';
import ElementNamedNodeMap from '../element/ElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class SVGElementNamedNodeMap extends ElementNamedNodeMap {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (item[PropertySymbol.name] === 'style' &&
            this[PropertySymbol.ownerElement][PropertySymbol.style]) {
            this[PropertySymbol.ownerElement][PropertySymbol.style].cssText = item[PropertySymbol.value];
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem &&
            removedItem[PropertySymbol.name] === 'style' &&
            this[PropertySymbol.ownerElement][PropertySymbol.style]) {
            this[PropertySymbol.ownerElement][PropertySymbol.style].cssText = '';
        }
        return removedItem;
    }
}
//# sourceMappingURL=SVGElementNamedNodeMap.js.map