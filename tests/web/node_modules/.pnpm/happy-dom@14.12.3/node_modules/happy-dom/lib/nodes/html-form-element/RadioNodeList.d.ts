import HTMLButtonElement from '../html-button-element/HTMLButtonElement.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';
import NodeList from '../node/NodeList.js';
/**
 * RadioNodeList
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RadioNodeList
 */
export default class RadioNodeList extends NodeList<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | HTMLButtonElement> {
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): string;
}
//# sourceMappingURL=RadioNodeList.d.ts.map