import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLInputElement from '../html-input-element/HTMLInputElement.cjs';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.cjs';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.cjs';
import RadioNodeList from './RadioNodeList.cjs';
import HTMLButtonElement from '../html-button-element/HTMLButtonElement.cjs';
/**
 * HTMLFormControlsCollection.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormControlsCollection
 */
export default class HTMLFormControlsCollection extends Array<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement> implements HTMLFormControlsCollection {
    [PropertySymbol.namedItems]: {
        [k: string]: RadioNodeList;
    };
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index: number): HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement | null;
    /**
     * Returns named item.
     *
     * @param name Name.
     * @returns Node.
     */
    namedItem(name: string): HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement | RadioNodeList | null;
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.appendNamedItem](node: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement, name: string): void;
    /**
     * Appends named item.
     *
     * @param node Node.
     * @param name Name.
     */
    [PropertySymbol.removeNamedItem](node: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement, name: string): void;
    /**
     * Returns "true" if the property name is valid.
     *
     * @param name Name.
     * @returns True if the property name is valid.
     */
    protected [PropertySymbol.isValidPropertyName](name: string): boolean;
}
//# sourceMappingURL=HTMLFormControlsCollection.d.ts.map