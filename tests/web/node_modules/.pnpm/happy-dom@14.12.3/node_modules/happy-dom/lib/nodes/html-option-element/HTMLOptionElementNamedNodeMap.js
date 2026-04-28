import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLOptionElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if (!this[PropertySymbol.ownerElement][PropertySymbol.dirtyness] &&
            item[PropertySymbol.name] === 'selected' &&
            replacedItem?.[PropertySymbol.value] !== item[PropertySymbol.value]) {
            const selectNode = (this[PropertySymbol.ownerElement][PropertySymbol.selectNode]);
            this[PropertySymbol.ownerElement][PropertySymbol.selectedness] = true;
            if (selectNode) {
                selectNode[PropertySymbol.updateOptionItems](this[PropertySymbol.ownerElement]);
            }
        }
        return replacedItem || null;
    }
    /**
     * @override
     */
    [(PropertySymbol.ownerElement, PropertySymbol.removeNamedItem)](name) {
        const removedItem = super[PropertySymbol.removeNamedItem](name);
        if (removedItem &&
            !this[PropertySymbol.ownerElement][PropertySymbol.dirtyness] &&
            removedItem[PropertySymbol.name] === 'selected') {
            const selectNode = (this[PropertySymbol.ownerElement][PropertySymbol.selectNode]);
            this[PropertySymbol.ownerElement][PropertySymbol.selectedness] = false;
            if (selectNode) {
                selectNode[PropertySymbol.updateOptionItems]();
            }
        }
        return removedItem;
    }
}
//# sourceMappingURL=HTMLOptionElementNamedNodeMap.js.map