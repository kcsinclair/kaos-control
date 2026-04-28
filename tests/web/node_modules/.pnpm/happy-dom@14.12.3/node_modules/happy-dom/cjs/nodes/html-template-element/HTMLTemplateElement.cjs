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
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const XMLSerializer_js_1 = __importDefault(require("../../xml-serializer/XMLSerializer.cjs"));
const XMLParser_js_1 = __importDefault(require("../../xml-parser/XMLParser.cjs"));
/**
 * HTML Template Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTemplateElement.
 */
class HTMLTemplateElement extends HTMLElement_js_1.default {
    constructor() {
        super(...arguments);
        // Internal properties
        this[_a] = this[PropertySymbol.ownerDocument].createDocumentFragment();
    }
    /**
     * Returns content.
     *
     * @returns Content.
     */
    get content() {
        return this[PropertySymbol.content];
    }
    /**
     * @override
     */
    get innerHTML() {
        return this.getInnerHTML();
    }
    /**
     * @override
     */
    set innerHTML(html) {
        const content = this[PropertySymbol.content];
        for (const child of content[PropertySymbol.childNodes].slice()) {
            this[PropertySymbol.content].removeChild(child);
        }
        XMLParser_js_1.default.parse(this[PropertySymbol.ownerDocument], html, {
            rootNode: this[PropertySymbol.content]
        });
    }
    /**
     * @override
     */
    get firstChild() {
        return this[PropertySymbol.content].firstChild;
    }
    /**
     * @override
     */
    get lastChild() {
        return this[PropertySymbol.content].lastChild;
    }
    /**
     * @override
     */
    getInnerHTML(options) {
        const xmlSerializer = new XMLSerializer_js_1.default({
            includeShadowRoots: options && options.includeShadowRoots,
            escapeEntities: false
        });
        const content = this[PropertySymbol.content];
        let xml = '';
        for (const node of content[PropertySymbol.childNodes]) {
            xml += xmlSerializer.serializeToString(node);
        }
        return xml;
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.content, PropertySymbol.appendChild)](node) {
        return this[PropertySymbol.content].appendChild(node);
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        return this[PropertySymbol.content].removeChild(node);
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        return this[PropertySymbol.content].insertBefore(newNode, referenceNode);
    }
    /**
     * @override
     */
    [PropertySymbol.replaceChild](newChild, oldChild) {
        return this[PropertySymbol.content].replaceChild(newChild, oldChild);
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.content] = this[PropertySymbol.content].cloneNode(deep);
        return clone;
    }
}
exports.default = HTMLTemplateElement;
//# sourceMappingURL=HTMLTemplateElement.cjs.map