import DOMRectReadOnly from './DOMRectReadOnly.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/* eslint-disable jsdoc/require-jsdoc */
/**
 * Bounding rect object.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRect
 */
export default class DOMRect extends DOMRectReadOnly {
    set x(value) {
        this[PropertySymbol.x] = value;
    }
    get x() {
        return this[PropertySymbol.x];
    }
    set y(value) {
        this[PropertySymbol.y] = value;
    }
    get y() {
        return this[PropertySymbol.y];
    }
    set width(value) {
        this[PropertySymbol.width] = value;
    }
    get width() {
        return this[PropertySymbol.width];
    }
    set height(value) {
        this[PropertySymbol.height] = value;
    }
    get height() {
        return this[PropertySymbol.height];
    }
    static fromRect(other) {
        return new DOMRect(other.x, other.y, other.width, other.height);
    }
}
//# sourceMappingURL=DOMRect.js.map