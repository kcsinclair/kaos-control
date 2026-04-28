import * as PropertySymbol from '../PropertySymbol.cjs';
import Attr from '../nodes/attr/Attr.cjs';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class NamedNodeMap {
    [index: number]: Attr;
    length: number;
    protected [PropertySymbol.namedItems]: {
        [k: string]: Attr;
    };
    /**
     * Returns string.
     *
     * @returns string.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    [Symbol.iterator](): IterableIterator<Attr>;
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): Attr | null;
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Item.
     */
    getNamedItem(name: string): Attr | null;
    /**
     * Returns item by name and namespace.
     *
     * @param namespace Namespace.
     * @param localName Local name of the attribute.
     * @returns Item.
     */
    getNamedItemNS(namespace: string, localName: string): Attr | null;
    /**
     * Sets named item.
     *
     * @param item Item.
     * @returns Replaced item.
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * Adds a new namespaced item.
     *
     * @alias setNamedItem()
     * @param item Item.
     * @returns Replaced item.
     */
    setNamedItemNS(item: Attr): Attr | null;
    /**
     * Removes an item.
     *
     * @throws DOMException
     * @param name Name of item.
     * @returns Removed item.
     */
    removeNamedItem(name: string): Attr;
    /**
     * Removes a namespaced item.
     *
     * @param namespace Namespace.
     * @param localName Local name of the item.
     * @returns Removed item.
     */
    removeNamedItemNS(namespace: string, localName: string): Attr | null;
    /**
     * Sets named item without calling listeners for certain attributes.
     *
     * @param item Item.
     * @returns Replaced item.
     */
    [PropertySymbol.setNamedItemWithoutConsequences](item: Attr): Attr | null;
    /**
     * Removes an item without throwing if it doesn't exist.
     *
     * @param name Name of item.
     * @returns Removed item, or null if it didn't exist.
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
    /**
     * Removes an item without calling listeners for certain attributes.
     *
     * @param name Name of item.
     * @returns Removed item, or null if it didn't exist.
     */
    [PropertySymbol.removeNamedItemWithoutConsequences](name: string): Attr | null;
    /**
     * Removes an item from index.
     *
     * @param item Item.
     */
    protected [PropertySymbol.removeNamedItemIndex](item: Attr): void;
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    protected [PropertySymbol.isValidPropertyName](name: string): boolean;
}
//# sourceMappingURL=NamedNodeMap.d.ts.map