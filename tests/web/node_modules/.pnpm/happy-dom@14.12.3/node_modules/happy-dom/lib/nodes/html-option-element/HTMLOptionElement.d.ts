import NamedNodeMap from '../../named-node-map/NamedNodeMap.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import Node from '../node/Node.js';
/**
 * HTML Option Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement.
 */
export default class HTMLOptionElement extends HTMLElement {
    [PropertySymbol.attributes]: NamedNodeMap;
    [PropertySymbol.selectedness]: boolean;
    [PropertySymbol.dirtyness]: boolean;
    /**
     * Returns inner text, which is the rendered appearance of text.
     *
     * @returns Inner text.
     */
    get text(): string;
    /**
     * Sets the inner text, which is the rendered appearance of text.
     *
     * @param innerText Inner text.
     */
    set text(text: string);
    /**
     * Returns index.
     *
     * @returns Index.
     */
    get index(): number;
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form(): HTMLFormElement;
    /**
     * Returns selected.
     *
     * @returns Selected.
     */
    get selected(): boolean;
    /**
     * Sets selected.
     *
     * @param selected Selected.
     */
    set selected(selected: boolean);
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled(): boolean;
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled: boolean);
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): string;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: string);
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
}
//# sourceMappingURL=HTMLOptionElement.d.ts.map