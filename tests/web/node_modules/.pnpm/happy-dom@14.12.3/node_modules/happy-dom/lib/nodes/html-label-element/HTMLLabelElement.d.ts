import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import Event from '../../event/Event.js';
/**
 * HTML Label Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLLabelElement.
 */
export default class HTMLLabelElement extends HTMLElement {
    cloneNode: (deep?: boolean) => HTMLLabelElement;
    /**
     * Returns a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @returns ID of the labeled control.
     */
    get htmlFor(): string;
    /**
     * Sets a string containing the ID of the labeled control. This reflects the "for" attribute.
     *
     * @param htmlFor ID of the labeled control.
     */
    set htmlFor(htmlFor: string);
    /**
     * Returns an HTML element representing the control with which the label is associated.
     *
     * @returns Control element.
     */
    get control(): HTMLElement;
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLLabelElement;
    /**
     * @override
     */
    dispatchEvent(event: Event): boolean;
}
//# sourceMappingURL=HTMLLabelElement.d.ts.map