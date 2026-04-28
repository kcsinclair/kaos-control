var _a, _b, _c;
import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import HTMLCollection from '../element/HTMLCollection.js';
import ElementUtility from '../element/ElementUtility.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * DocumentFragment.
 */
export default class DocumentFragment extends Node {
    constructor() {
        super(...arguments);
        this[_a] = new HTMLCollection();
        this[_b] = this;
        this[_c] = NodeTypeEnum.documentFragmentNode;
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
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode ||
                childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
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
        ParentNodeUtility.append(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    prepend(...nodes) {
        ParentNodeUtility.prepend(this, ...nodes);
    }
    /**
     * Replaces the existing children of a node with a specified new set of children.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceChildren(...nodes) {
        ParentNodeUtility.replaceChildren(this, ...nodes);
    }
    /**
     * Query CSS selector to find matching elments.
     *
     * @param selector CSS selector.
     * @returns Matching elements.
     */
    querySelectorAll(selector) {
        return QuerySelector.querySelectorAll(this, selector);
    }
    /**
     * Query CSS Selector to find a matching element.
     *
     * @param selector CSS selector.
     * @returns Matching element.
     */
    querySelector(selector) {
        return QuerySelector.querySelector(this, selector);
    }
    /**
     * Returns an element by ID.
     *
     * @param id ID.
     * @returns Matching element.
     */
    getElementById(id) {
        return ParentNodeUtility.getElementById(this, id);
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.children, _b = PropertySymbol.rootNode, _c = PropertySymbol.nodeType, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        if (deep) {
            for (const node of clone[PropertySymbol.childNodes]) {
                if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
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
        return ElementUtility.appendChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility.removeChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility.insertBefore(this, newNode, referenceNode);
    }
}
//# sourceMappingURL=DocumentFragment.js.map