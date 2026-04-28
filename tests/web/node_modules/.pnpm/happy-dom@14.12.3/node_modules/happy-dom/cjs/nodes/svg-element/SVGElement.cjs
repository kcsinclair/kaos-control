"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _SVGElement_dataset, _a, _b;
Object.defineProperty(exports, "__esModule", { value: true });
const CSSStyleDeclaration_js_1 = __importDefault(require("../../css/declaration/CSSStyleDeclaration.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const Element_js_1 = __importDefault(require("../element/Element.cjs"));
const HTMLElementUtility_js_1 = __importDefault(require("../html-element/HTMLElementUtility.cjs"));
const SVGElementNamedNodeMap_js_1 = __importDefault(require("./SVGElementNamedNodeMap.cjs"));
const DatasetFactory_js_1 = __importDefault(require("../element/DatasetFactory.cjs"));
/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
class SVGElement extends Element_js_1.default {
    constructor() {
        super(...arguments);
        // Events
        this.onabort = null;
        this.onerror = null;
        this.onload = null;
        this.onresize = null;
        this.onscroll = null;
        this.onunload = null;
        // Internal properties
        this[_a] = new SVGElementNamedNodeMap_js_1.default(this);
        this[_b] = null;
        // Private properties
        _SVGElement_dataset.set(this, null);
    }
    /**
     * Returns viewport.
     *
     * @returns SVG rect.
     */
    get viewportElement() {
        return null;
    }
    /**
     * Returns current translate.
     *
     * @returns Element.
     */
    get ownerSVGElement() {
        let parent = this[PropertySymbol.parentNode];
        while (parent) {
            if (parent[PropertySymbol.localName] === 'svg') {
                return parent;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return null;
    }
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset() {
        return (__classPrivateFieldSet(this, _SVGElement_dataset, __classPrivateFieldGet(this, _SVGElement_dataset, "f") ?? DatasetFactory_js_1.default.createDataset(this), "f"));
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this[PropertySymbol.style]) {
            this[PropertySymbol.style] = new CSSStyleDeclaration_js_1.default(this);
        }
        return this[PropertySymbol.style];
    }
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex() {
        const tabIndex = this.getAttribute('tabindex');
        return tabIndex !== null ? Number(tabIndex) : -1;
    }
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex) {
        if (tabIndex === -1) {
            this.removeAttribute('tabindex');
        }
        else {
            this.setAttribute('tabindex', String(tabIndex));
        }
    }
    /**
     * Triggers a blur event.
     */
    blur() {
        HTMLElementUtility_js_1.default.blur(this);
    }
    /**
     * Triggers a focus event.
     */
    focus() {
        HTMLElementUtility_js_1.default.focus(this);
    }
}
_SVGElement_dataset = new WeakMap(), _a = PropertySymbol.attributes, _b = PropertySymbol.style;
exports.default = SVGElement;
//# sourceMappingURL=SVGElement.cjs.map