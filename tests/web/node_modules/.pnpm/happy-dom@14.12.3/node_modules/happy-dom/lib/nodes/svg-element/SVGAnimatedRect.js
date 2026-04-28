import SVGRect from './SVGRect.js';
/**
 * Rect object.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedRect
 */
export default class SVGAnimatedRect {
    constructor() {
        this.baseVal = new SVGRect();
        this.animVal = new SVGRect();
    }
}
//# sourceMappingURL=SVGAnimatedRect.js.map