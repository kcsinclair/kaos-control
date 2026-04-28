import HTMLButtonElement from '../html-button-element/HTMLButtonElement.cjs';
import HTMLInputElement from '../html-input-element/HTMLInputElement.cjs';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.cjs';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.cjs';
import NodeList from '../node/NodeList.cjs';
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