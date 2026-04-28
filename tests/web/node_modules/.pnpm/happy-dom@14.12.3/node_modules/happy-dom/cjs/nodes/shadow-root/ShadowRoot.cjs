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
const DocumentFragment_js_1 = __importDefault(require("../document-fragment/DocumentFragment.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const XMLParser_js_1 = __importDefault(require("../../xml-parser/XMLParser.cjs"));
const XMLSerializer_js_1 = __importDefault(require("../../xml-serializer/XMLSerializer.cjs"));
/**
 * ShadowRoot.
 */
class ShadowRoot extends DocumentFragment_js_1.default {
    constructor() {
        super(...arguments);
        // Events
        this.onslotchange = null;
        // Internal properties
        this[_a] = [];
        this[_b] = 'open';
        this[_c] = null;
    }
    /**
     * Returns mode.
     *
     * @returns Mode.
     */
    get mode() {
        return this[PropertySymbol.mode];
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        return this[PropertySymbol.host];
    }
    /**
     * Returns inner HTML.
     *
     * @returns HTML.
     */
    get innerHTML() {
        const xmlSerializer = new XMLSerializer_js_1.default({
            escapeEntities: false
        });
        let xml = '';
        for (const node of this[PropertySymbol.childNodes]) {
            xml += xmlSerializer.serializeToString(node);
        }
        return xml;
    }
    /**
     * Sets inner HTML.
     *
     * @param html HTML.
     */
    set innerHTML(html) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        XMLParser_js_1.default.parse(this[PropertySymbol.ownerDocument], html, { rootNode: this });
    }
    /**
     * Returns adopted style sheets.
     *
     * @returns Adopted style sheets.
     */
    get adoptedStyleSheets() {
        return this[PropertySymbol.adoptedStyleSheets];
    }
    /**
     * Sets adopted style sheets.
     *
     * @param value Adopted style sheets.
     */
    set adoptedStyleSheets(value) {
        this[PropertySymbol.adoptedStyleSheets] = value;
    }
    /**
     * Returns active element.
     *
     * @returns Active element.
     */
    get activeElement() {
        const activeElement = this[PropertySymbol.ownerDocument][PropertySymbol.activeElement];
        if (activeElement &&
            activeElement[PropertySymbol.isConnected] &&
            activeElement.getRootNode() === this) {
            return activeElement;
        }
        return null;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return this.innerHTML;
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.adoptedStyleSheets, _b = PropertySymbol.mode, _c = PropertySymbol.host, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.mode] = this.mode;
        return clone;
    }
}
exports.default = ShadowRoot;
//# sourceMappingURL=ShadowRoot.cjs.map