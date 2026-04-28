import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * HTML collection.
 */
export default class HTMLCollection<T> extends Array implements HTMLCollection<T> {
    protected [PropertySymbol.namedItems]: {
        [k: string]: T[];
    };
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): T | null;
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name: string): T | null;
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.appendNamedItem](node: T, name: string): void;
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeNamedItem](node: T, name: string): void;
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    protected [PropertySymbol.isValidPropertyName](name: string): boolean;
}
//# sourceMappingURL=HTMLCollection.d.ts.map