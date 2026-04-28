import HTMLCollection from '../element/HTMLCollection.cjs';
import HTMLSelectElement from './HTMLSelectElement.cjs';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.cjs';
/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
export default class HTMLOptionsCollection extends HTMLCollection<HTMLOptionElement> {
    #private;
    /**
     *
     * @param selectElement
     */
    constructor(selectElement: HTMLSelectElement);
    /**
     * Returns selectedIndex.
     *
     * @returns SelectedIndex.
     */
    get selectedIndex(): number;
    /**
     * Sets selectedIndex.
     *
     * @param selectedIndex SelectedIndex.
     */
    set selectedIndex(selectedIndex: number);
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): HTMLOptionElement;
    /**
     *
     * @param element
     * @param before
     */
    add(element: HTMLOptionElement, before?: number | HTMLOptionElement): void;
    /**
     * Removes indexed element from collection.
     *
     * @param index Index.
     */
    remove(index: number): void;
}
//# sourceMappingURL=HTMLOptionsCollection.d.ts.map