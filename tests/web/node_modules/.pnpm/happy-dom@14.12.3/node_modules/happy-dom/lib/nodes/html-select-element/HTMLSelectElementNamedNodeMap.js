import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLSelectElementNamedNodeMap extends HTMLElementNamedNodeMap {
    /**
     * @override
     */
    setNamedItem(item) {
        const replacedItem = super.setNamedItem(item);
        if ((item[PropertySymbol.name] === 'id' || item[PropertySymbol.name] === 'name') &&
            this[PropertySymbol.ownerElement][PropertySymbol.formNode]) {
            if (replacedItem && replacedItem[PropertySymbol.value]) {
                this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.removeFormControlItem](this[PropertySymbol.ownerElement], replacedItem[PropertySymbol.value]);
            }
            if (item[PropertySymbol.value]) {
                this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.appendFormControlItem](this[PropertySymbol.ownerElement], item[PropertySymbol.value]);
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
            (removedItem[PropertySymbol.name] === 'id' || removedItem[PropertySymbol.name] === 'name') &&
            this[PropertySymbol.ownerElement][PropertySymbol.formNode]) {
            this[PropertySymbol.ownerElement][PropertySymbol.formNode][PropertySymbol.removeFormControlItem](this[PropertySymbol.ownerElement], removedItem[PropertySymbol.value]);
        }
        return removedItem;
    }
}
//# sourceMappingURL=HTMLSelectElementNamedNodeMap.js.map