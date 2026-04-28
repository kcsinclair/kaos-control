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
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const QuerySelector_js_1 = __importDefault(require("../../query-selector/QuerySelector.cjs"));
const ParentNodeUtility_js_1 = __importDefault(require("../parent-node/ParentNodeUtility.cjs"));
const HTMLCollection_js_1 = __importDefault(require("../element/HTMLCollection.cjs"));
const ElementUtility_js_1 = __importDefault(require("../element/ElementUtility.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * DocumentFragment.
 */
class DocumentFragment extends Node_js_1.default {
    constructor() {
        super(...arguments);
        this[_a] = new HTMLCollection_js_1.default();
        this[_b] = this;
        this[_c] = NodeTypeEnum_js_1.default.documentFragmentNode;
    }
    /**
     * Returns the document fragment children.
     */
    get children() {
        return this[PropertySymbol.children];
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get childElementCount() {
        return this[PropertySymbol.children].length;
    }
    /**
     * First element child.
     *
     * @returns Element.
     */
    get firstElementChild() {
        return this[PropertySymbol.children][0] ?? null;
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get lastElementChild() {
        return this[PropertySymbol.children][this[PropertySymbol.children].length - 1] ?? null;
    }
    /**
     * Get text value of children.
     *
     * @returns Text content.
     */
    get textContent() {
        let result = '';
        for (const childNode of this[PropertySymbol.childNodes]) {
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.elementNode ||
                childNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.textNode) {
                result += childNode.textContent;
            }
        }
        return result;
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        if (textContent) {
            this.appendChild(this[PropertySymbol.ownerDocument].createTextNode(textContent));
        }
    }
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    append(...nodes) {
        ParentNodeUtility_js_1.default.append(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    prepend(...nodes) {
        ParentNodeUtility_js_1.default.prepend(this, ...nodes);
    }
    /**
     * Replaces the existing children of a node with a specified new set of children.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceChildren(...nodes) {
        ParentNodeUtility_js_1.default.replaceChildren(this, ...nodes);
    }
    /**
     * Query CSS selector to find matching elments.
     *
     * @param selector CSS selector.
     * @returns Matching elements.
     */
    querySelectorAll(selector) {
        return QuerySelector_js_1.default.querySelectorAll(this, selector);
    }
    /**
     * Query CSS Selector to find a matching element.
     *
     * @param selector CSS selector.
     * @returns Matching element.
     */
    querySelector(selector) {
        return QuerySelector_js_1.default.querySelector(this, selector);
    }
    /**
     * Returns an element by ID.
     *
     * @param id ID.
     * @returns Matching element.
     */
    getElementById(id) {
        return ParentNodeUtility_js_1.default.getElementById(this, id);
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.children, _b = PropertySymbol.rootNode, _c = PropertySymbol.nodeType, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        if (deep) {
            for (const node of clone[PropertySymbol.childNodes]) {
                if (node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.elementNode) {
                    clone[PropertySymbol.children].push(node);
                }
            }
        }
        return clone;
    }
    /**
     * @override
     */
    [PropertySymbol.appendChild](node) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility_js_1.default.appendChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility_js_1.default.removeChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility_js_1.default.insertBefore(this, newNode, referenceNode);
    }
}
exports.default = DocumentFragment;
//# sourceMappingURL=DocumentFragment.cjs.map