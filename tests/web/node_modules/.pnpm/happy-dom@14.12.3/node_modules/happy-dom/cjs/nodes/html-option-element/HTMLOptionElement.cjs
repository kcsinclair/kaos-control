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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _a, _b, _c;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const HTMLOptionElementNamedNodeMap_js_1 = __importDefault(require("./HTMLOptionElementNamedNodeMap.cjs"));
/**
 * HTML Option Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement.
 */
class HTMLOptionElement extends HTMLElement_js_1.default {
    constructor() {
        super(...arguments);
        this[_a] = new HTMLOptionElementNamedNodeMap_js_1.default(this);
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
exports.default = HTMLOptionElement;
//# sourceMappingURL=HTMLOptionElement.cjs.map