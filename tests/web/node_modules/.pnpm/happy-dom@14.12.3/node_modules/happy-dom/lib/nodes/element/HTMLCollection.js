var _a;
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML collection.
 */
export default class HTMLCollection extends Array {
    constructor() {
        super(...arguments);
        this[_a] = {};
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index) {
        return index >= 0 && this[index] ? this[index] : null;
    }
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name) {
        return this[PropertySymbol.namedItems][name] && this[PropertySymbol.namedItems][name].length
            ? this[PropertySymbol.namedItems][name][0]
            : null;
    }
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [(_a = PropertySymbol.namedItems, PropertySymbol.appendNamedItem)](node, name) {
        if (name) {
            this[PropertySymbol.namedItems][name] = this[PropertySymbol.namedItems][name] || [];
            if (!this[PropertySymbol.namedItems][name].includes(node)) {
                this[PropertySymbol.namedItems][name].push(node);
            }
            if (!this.hasOwnProperty(name) && this[PropertySymbol.isValidPropertyName](name)) {
                this[name] = this[PropertySymbol.namedItems][name][0];
            }
        }
    }
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeNamedItem](node, name) {
        if (name && this[PropertySymbol.namedItems][name]) {
            const index = this[PropertySymbol.namedItems][name].indexOf(node);
            if (index > -1) {
                this[PropertySymbol.namedItems][name].splice(index, 1);
                if (this[PropertySymbol.namedItems][name].length === 0) {
                    delete this[PropertySymbol.namedItems][name];
                    if (this.hasOwnProperty(name) && this[PropertySymbol.isValidPropertyName](name)) {
                        delete this[name];
                    }
                }
                else if (this[PropertySymbol.isValidPropertyName](name)) {
                    this[name] = this[PropertySymbol.namedItems][name][0];
                }
            }
        }
    }
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    [PropertySymbol.isValidPropertyName](name) {
        return (!!name &&
            !this.constructor.prototype.hasOwnProperty(name) &&
            !Array.prototype.hasOwnProperty(name) &&
            (isNaN(Number(name)) || name.includes('.')));
    }
}
//# sourceMappingURL=HTMLCollection.js.map