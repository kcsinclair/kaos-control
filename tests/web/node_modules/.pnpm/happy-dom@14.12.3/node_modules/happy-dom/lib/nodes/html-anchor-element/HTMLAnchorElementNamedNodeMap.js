import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLAnchorElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (item[PropertySymbol.name] === 'rel' &&
            this[PropertySymbol.ownerElement][PropertySymbol.relList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.relList][PropertySymbol.updateIndices]();
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem?.[PropertySymbol.name] === 'rel' &&
            this[PropertySymbol.ownerElement][PropertySymbol.relList]) {
            this[PropertySymbol.ownerElement][PropertySymbol.relList][PropertySymbol.updateIndices]();
        }
        return removedItem;
    }
}
//# sourceMappingURL=HTMLAnchorElementNamedNodeMap.js.map