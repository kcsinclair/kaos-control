import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Time Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTimeElement
 */
export default class HTMLTimeElement extends HTMLElement {
    /**
     * Returns dateTime.
     *
     * @returns dateTime.
     */
    get dateTime() {
        return this.getAttribute('dateTime') || '';
    }
    /**
     * Sets dateTime.
     *
     * @param dateTime dateTime.
     */
    set dateTime(dateTime) {
        this.setAttribute('dateTime', dateTime);
    }
}
//# sourceMappingURL=HTMLTimeElement.js.map