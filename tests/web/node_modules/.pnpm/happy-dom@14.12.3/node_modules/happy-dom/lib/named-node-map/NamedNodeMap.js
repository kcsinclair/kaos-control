var _a;
import * as PropertySymbol from '../PropertySymbol.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class NamedNodeMap {
    constructor() {
        this.length = 0;
        this[_a] = {};
    }
    /**
     * Returns string.
     *
     * @returns string.
     */
    get [(_a = PropertySymbol.namedItems, Symbol.toStringTag)]() {
        return 'NamedNodeMap';
    }
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    *[Symbol.iterator]() {
        for (let i = 0, max = this.length; i < max; i++) {
            yield this[i];
        }
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
     * @returns Item.
     */
    getNamedItem(name) {
        return this[PropertySymbol.namedItems][name] || null;
    }
    /**
     * Returns item by name and namespace.
     *
     * @param namespace Namespace.
     * @param localName Local name of the attribute.
     * @returns Item.
     */
    getNamedItemNS(namespace, localName) {
        const attribute = this.getNamedItem(localName);
        if (attribute &&
            attribute[PropertySymbol.namespaceURI] === namespace &&
            attribute.localName === localName) {
            return attribute;
        }
        for (let i = 0, max = this.length; i < max; i++) {
            if (this[i][PropertySymbol.namespaceURI] === namespace && this[i].localName === localName) {
                return this[i];
            }
        }
        return null;
    }
    /**
     * Sets named item.
     *
     * @param item Item.
     * @returns Replaced item.
     */
    setNamedItem(item) {
        return this[PropertySymbol.setNamedItemWithoutConsequences](item);
    }
    /**
     * Adds a new namespaced item.
     *
     * @alias setNamedItem()
     * @param item Item.
     * @returns Replaced item.
     */
    setNamedItemNS(item) {
        return this.setNamedItem(item);
    }
    /**
     * Removes an item.
     *
     * @throws DOMException
     * @param name Name of item.
     * @returns Removed item.
     */
    removeNamedItem(name) {
        const item = this[PropertySymbol.removeNamedItem](name);
        if (!item) {
            throw new DOMException(`Failed to execute 'removeNamedItem' on 'NamedNodeMap': No item with name '${name}' was found.`, DOMExceptionNameEnum.notFoundError);
        }
        return item;
    }
    /**
     * Removes a namespaced item.
     *
     * @param namespace Namespace.
     * @param localName Local name of the item.
     * @returns Removed item.
     */
    removeNamedItemNS(namespace, localName) {
        const attribute = this.getNamedItemNS(namespace, localName);
        if (attribute) {
            return this.removeNamedItem(attribute[PropertySymbol.name]);
        }
        return null;
    }
    /**
     * Sets named item without calling listeners for certain attributes.
     *
     * @param item Item.
     * @returns Replaced item.
     */
    [PropertySymbol.setNamedItemWithoutConsequences](item) {
        if (item[PropertySymbol.name]) {
            const replacedItem = this[PropertySymbol.namedItems][item[PropertySymbol.name]] || null;
            this[PropertySymbol.namedItems][item[PropertySymbol.name]] = item;
            if (replacedItem) {
                this[PropertySymbol.removeNamedItemIndex](replacedItem);
            }
            this[this.length] = item;
            this.length++;
            if (this[PropertySymbol.isValidPropertyName](item[PropertySymbol.name])) {
                this[item[PropertySymbol.name]] = item;
            }
            return replacedItem;
        }
        return null;
    }
    /**
     * Removes an item without throwing if it doesn't exist.
     *
     * @param name Name of item.
     * @returns Removed item, or null if it didn't exist.
     */
    [PropertySymbol.removeNamedItem](name) {
        return this[PropertySymbol.removeNamedItemWithoutConsequences](name);
    }
    /**
     * Removes an item without calling listeners for certain attributes.
     *
     * @param name Name of item.
     * @returns Removed item, or null if it didn't exist.
     */
    [PropertySymbol.removeNamedItemWithoutConsequences](name) {
        const removedItem = this[PropertySymbol.namedItems][name] || null;
        if (!removedItem) {
            return null;
        }
        this[PropertySymbol.removeNamedItemIndex](removedItem);
        if (this[name] === removedItem) {
            delete this[name];
        }
        delete this[PropertySymbol.namedItems][name];
        return removedItem;
    }
    /**
     * Removes an item from index.
     *
     * @param item Item.
     */
    [PropertySymbol.removeNamedItemIndex](item) {
        for (let i = 0; i < this.length; i++) {
            if (this[i] === item) {
                for (let b = i; b < this.length; b++) {
                    if (b < this.length - 1) {
                        this[b] = this[b + 1];
                    }
                    else {
                        delete this[b];
                    }
                }
                this.length--;
                break;
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
            (isNaN(Number(name)) || name.includes('.')));
    }
}
//# sourceMappingURL=NamedNodeMap.js.map