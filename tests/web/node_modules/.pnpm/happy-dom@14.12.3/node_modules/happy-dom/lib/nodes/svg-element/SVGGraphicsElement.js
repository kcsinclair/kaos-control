import SVGElement from './SVGElement.js';
import DOMRect from '../element/DOMRect.js';
import DOMMatrix from './DOMMatrix.js';
/**
 * SVG Graphics Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGraphicsElement
 */
export default class SVGGraphicsElement extends SVGElement {
    constructor() {
        super(...arguments);
        this.transform = {};
    }
    /**
     * Returns DOM rect.
     *
     * @returns DOM rect.
     */
    getBBox() {
        return new DOMRect();
    }
    /**
     * Returns CTM.
     *
     * @returns CTM.
     */
    getCTM() {
        return new DOMMatrix();
    }
    /**
     * Returns screen CTM.
     *
     * @returns Screen CTM.
     */
    getScreenCTM() {
        return new DOMMatrix();
    }
}
//# sourceMappingURL=SVGGraphicsElement.js.map