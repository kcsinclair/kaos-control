var _a, _b;
import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTML Style Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLStyleElement.
 */
export default class HTMLStyleElement extends HTMLElement {
    constructor() {
        super(...arguments);
        this[_a] = null;
        this[_b] = this;
    }
    /**
     * Returns CSS style sheet.
     *
     * @returns CSS style sheet.
     */
    get sheet() {
        return this[PropertySymbol.sheet] ? this[PropertySymbol.sheet] : null;
    }
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || '';
    }
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media) {
        this.setAttribute('media', media);
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
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
     * @override
     */
    [(_a = PropertySymbol.sheet, _b = PropertySymbol.styleNode, PropertySymbol.appendChild)](node) {
        const returnValue = super[PropertySymbol.appendChild](node);
        this[PropertySymbol.updateSheet]();
        return returnValue;
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        const returnValue = super[PropertySymbol.removeChild](node);
        this[PropertySymbol.updateSheet]();
        return returnValue;
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        const returnValue = super[PropertySymbol.insertBefore](newNode, referenceNode);
        this[PropertySymbol.updateSheet]();
        return returnValue;
    }
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        super[PropertySymbol.connectToNode](parentNode);
        if (parentNode) {
            this[PropertySymbol.sheet] = new CSSStyleSheet();
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
        else {
            this[PropertySymbol.sheet] = null;
        }
    }
    /**
     * Updates the CSSStyleSheet with the text content.
     */
    [PropertySymbol.updateSheet]() {
        if (this[PropertySymbol.sheet]) {
            this[PropertySymbol.ownerDocument][PropertySymbol.cacheID]++;
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
    }
}
//# sourceMappingURL=HTMLStyleElement.js.map