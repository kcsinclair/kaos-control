"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const SVGElement_js_1 = __importDefault(require("./SVGElement.cjs"));
const DOMRect_js_1 = __importDefault(require("../element/DOMRect.cjs"));
const DOMMatrix_js_1 = __importDefault(require("./DOMMatrix.cjs"));
/**
 * SVG Graphics Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGraphicsElement
 */
class SVGGraphicsElement extends SVGElement_js_1.default {
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
        return new DOMRect_js_1.default();
    }
    /**
     * Returns CTM.
     *
     * @returns CTM.
     */
    getCTM() {
        return new DOMMatrix_js_1.default();
    }
    /**
     * Returns screen CTM.
     *
     * @returns Screen CTM.
     */
    getScreenCTM() {
        return new DOMMatrix_js_1.default();
    }
}
exports.default = SVGGraphicsElement;
//# sourceMappingURL=SVGGraphicsElement.cjs.map