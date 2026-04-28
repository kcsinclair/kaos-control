var _a, _b, _c;
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLOptionElementNamedNodeMap from './HTMLOptionElementNamedNodeMap.js';
/**
 * HTML Option Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement.
 */
export default class HTMLOptionElement extends HTMLElement {
    constructor() {
        super(...arguments);
        this[_a] = new HTMLOptionElementNamedNodeMap(this);
        this[_b] = false;
        this[_c] = false;
    }
    /**
     * Returns inner text, which is the rendered appearance of text.
     *
     * @returns Inner text.
     */
    get text() {
        return this.innerText;
    }
    /**
     * Sets the inner text, which is the rendered appearance of text.
     *
     * @param innerText Inner text.
     */
    set text(text) {
        this.innerText = text;
    }
    /**
     * Returns index.
     *
     * @returns Index.
     */
    get index() {
        return this[PropertySymbol.selectNode]
            ? this[PropertySymbol.selectNode].options.indexOf(this)
            : 0;
    }
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        return this[PropertySymbol.formNode];
    }
    /**
     * Returns selected.
     *
     * @returns Selected.
     */
    get selected() {
        return this[PropertySymbol.selectedness];
    }
    /**
     * Sets selected.
     *
     * @param selected Selected.
     */
    set selected(selected) {
        const selectNode = this[PropertySymbol.selectNode];
        this[PropertySymbol.dirtyness] = true;
        this[PropertySymbol.selectedness] = Boolean(selected);
        if (selectNode) {
            selectNode[PropertySymbol.updateOptionItems](this[PropertySymbol.selectedness] ? this : null);
        }
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this.getAttribute('disabled') !== null;
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        if (!disabled) {
            this.removeAttribute('disabled');
        }
        else {
            this.setAttribute('disabled', '');
        }
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this.getAttribute('value') ?? this.textContent;
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.setAttribute('value', value);
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.attributes, _b = PropertySymbol.selectedness, _c = PropertySymbol.dirtyness, PropertySymbol.connectToNode)](parentNode = null) {
        const oldSelectNode = this[PropertySymbol.selectNode];
        super[PropertySymbol.connectToNode](parentNode);
        if (oldSelectNode !== this[PropertySymbol.selectNode]) {
            if (oldSelectNode) {
                oldSelectNode[PropertySymbol.updateOptionItems]();
            }
            if (this[PropertySymbol.selectNode]) {
                this[PropertySymbol.selectNode][PropertySymbol.updateOptionItems]();
            }
        }
    }
}
//# sourceMappingURL=HTMLOptionElement.js.map