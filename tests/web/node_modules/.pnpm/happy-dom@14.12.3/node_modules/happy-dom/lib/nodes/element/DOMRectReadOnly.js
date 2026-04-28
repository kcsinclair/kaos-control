var _a, _b, _c, _d;
import * as PropertySymbol from '../../PropertySymbol.js';
/* eslint-disable jsdoc/require-jsdoc */
/**
 * Bounding rect readonly object.
 *
 * @see https://drafts.fxtf.org/geometry/#DOMRect
 */
class DOMRectReadOnly {
    /**
     * Constructor.
     *
     * @param [x] X position.
     * @param [y] Y position.
     * @param [width] Width.
     * @param [height] Height.
     */
    constructor(x, y, width, height) {
        this[_a] = 0;
        this[_b] = 0;
        this[_c] = 0;
        this[_d] = 0;
        this[PropertySymbol.x] = x !== undefined && x !== null ? Number(x) : 0;
        this[PropertySymbol.y] = y !== undefined && y !== null ? Number(y) : 0;
        this[PropertySymbol.width] = width !== undefined && width !== null ? Number(width) : 0;
        this[PropertySymbol.height] = height !== undefined && height !== null ? Number(height) : 0;
    }
    get x() {
        return this[PropertySymbol.x];
    }
    get y() {
        return this[PropertySymbol.y];
    }
    get width() {
        return this[PropertySymbol.width];
    }
    get height() {
        return this[PropertySymbol.height];
    }
    get top() {
        return Math.min(this[PropertySymbol.y], this[PropertySymbol.y] + this[PropertySymbol.height]);
    }
    get right() {
        return Math.max(this[PropertySymbol.x], this[PropertySymbol.x] + this[PropertySymbol.width]);
    }
    get bottom() {
        return Math.max(this[PropertySymbol.y], this[PropertySymbol.y] + this[PropertySymbol.height]);
    }
    get left() {
        return Math.min(this[PropertySymbol.x], this[PropertySymbol.x] + this[PropertySymbol.width]);
    }
    toJSON() {
        return {
            x: this.x,
            y: this.y,
            width: this.width,
            height: this.height,
            top: this.top,
            right: this.right,
            bottom: this.bottom,
            left: this.left
        };
    }
    static fromRect(other) {
        return new DOMRectReadOnly(other.x, other.y, other.width, other.height);
    }
}
_a = PropertySymbol.x, _b = PropertySymbol.y, _c = PropertySymbol.width, _d = PropertySymbol.height;
export default DOMRectReadOnly;
//# sourceMappingURL=DOMRectReadOnly.js.map