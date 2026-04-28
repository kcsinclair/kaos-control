var _a, _b, _c, _d, _e, _f, _g, _h, _j;
import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from './NodeTypeEnum.js';
import NodeDocumentPositionEnum from './NodeDocumentPositionEnum.js';
import NodeUtility from './NodeUtility.js';
import NodeList from './NodeList.js';
import NodeFactory from '../NodeFactory.js';
/**
 * Node.
 */
class Node extends EventTarget {
    /**
     * Constructor.
     */
    constructor() {
        super();
        // Internal properties
        this[_a] = false;
        this[_b] = null;
        this[_c] = null;
        this[_d] = null;
        this[_e] = null;
        this[_f] = null;
        this[_g] = null;
        this[_h] = [];
        this[_j] = new NodeList();
        if (this.constructor[PropertySymbol.ownerDocument] !== undefined) {
            this[PropertySymbol.ownerDocument] = this.constructor[PropertySymbol.ownerDocument];
        }
        else {
            const ownerDocument = NodeFactory.pullOwnerDocument();
            if (!ownerDocument) {
                throw new Error('Failed to construct "Node": No owner document in queue. Please use "NodeFactory" to create instances of a Node.');
            }
            this[PropertySymbol.ownerDocument] = ownerDocument;
        }
    }
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [(PropertySymbol.ownerDocument, _a = PropertySymbol.isConnected, PropertySymbol.ownerDocument, _b = PropertySymbol.parentNode, PropertySymbol.nodeType, _c = PropertySymbol.rootNode, _d = PropertySymbol.formNode, _e = PropertySymbol.selectNode, _f = PropertySymbol.textAreaNode, _g = PropertySymbol.styleNode, _h = PropertySymbol.observers, _j = PropertySymbol.childNodes, Symbol.toStringTag)]() {
        return this.constructor.name;
    }
    /**
     * Returns connected state.
     *
     * @returns Connected state.
     */
    get isConnected() {
        return this[PropertySymbol.isConnected];
    }
    /**
     * Returns owner document.
     *
     * @returns Owner document.
     */
    get ownerDocument() {
        return this[PropertySymbol.ownerDocument];
    }
    /**
     * Returns parent node.
     *
     * @returns Parent node.
     */
    get parentNode() {
        return this[PropertySymbol.parentNode];
    }
    /**
     * Returns node type.
     *
     * @returns Node type.
     */
    get nodeType() {
        return this[PropertySymbol.nodeType];
    }
    /**
     * Get child nodes.
     *
     * @returns Child nodes list.
     */
    get childNodes() {
        return this[PropertySymbol.childNodes];
    }
    /**
     * Get text value of children.
     *
     * @returns Text content.
     */
    get textContent() {
        // Sub-classes should implement this method.
        return null;
    }
    /**
     * Sets text content.
     *
     * @param _textContent Text content.
     */
    set textContent(_textContent) {
        // Do nothing.
        // Sub-classes should implement this method.
    }
    /**
     * Node value.
     *
     * @returns Node value.
     */
    get nodeValue() {
        return null;
    }
    /**
     * Sets node value.
     */
    set nodeValue(_nodeValue) {
        // Do nothing
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return '';
    }
    /**
     * Previous sibling.
     *
     * @returns Node.
     */
    get previousSibling() {
        if (this[PropertySymbol.parentNode]) {
            const index = this[PropertySymbol.parentNode][PropertySymbol.childNodes].indexOf(this);
            if (index > 0) {
                return this[PropertySymbol.parentNode][PropertySymbol.childNodes][index - 1];
            }
        }
        return null;
    }
    /**
     * Next sibling.
     *
     * @returns Node.
     */
    get nextSibling() {
        if (this[PropertySymbol.parentNode]) {
            const index = this[PropertySymbol.parentNode][PropertySymbol.childNodes].indexOf(this);
            if (index > -1 &&
                index + 1 < this[PropertySymbol.parentNode][PropertySymbol.childNodes].length) {
                return this[PropertySymbol.parentNode][PropertySymbol.childNodes][index + 1];
            }
        }
        return null;
    }
    /**
     * First child.
     *
     * @returns Node.
     */
    get firstChild() {
        if (this[PropertySymbol.childNodes].length > 0) {
            return this[PropertySymbol.childNodes][0];
        }
        return null;
    }
    /**
     * Last child.
     *
     * @returns Node.
     */
    get lastChild() {
        if (this[PropertySymbol.childNodes].length > 0) {
            return this[PropertySymbol.childNodes][this[PropertySymbol.childNodes].length - 1];
        }
        return null;
    }
    /**
     * Returns parent element.
     *
     * @returns Element.
     */
    get parentElement() {
        let parent = this[PropertySymbol.parentNode];
        while (parent && parent[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
            parent = parent[PropertySymbol.parentNode];
        }
        return parent;
    }
    /**
     * Returns base URI.
     *
     * @returns Base URI.
     */
    get baseURI() {
        const base = this[PropertySymbol.ownerDocument].querySelector('base');
        if (base) {
            return base.href;
        }
        return this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].location.href;
    }
    /**
     * Returns "true" if the node has child nodes.
     *
     * @returns "true" if the node has child nodes.
     */
    hasChildNodes() {
        return this[PropertySymbol.childNodes].length > 0;
    }
    /**
     * Returns "true" if this node contains the other node.
     *
     * @param otherNode Node to test with.
     * @returns "true" if this node contains the other node.
     */
    contains(otherNode) {
        if (otherNode === undefined) {
            return false;
        }
        return NodeUtility.isInclusiveAncestor(this, otherNode);
    }
    /**
     * Returns closest root node (Document or ShadowRoot).
     *
     * @param options Options.
     * @param options.composed A Boolean that indicates whether the shadow root should be returned (false, the default), or a root node beyond shadow root (true).
     * @returns Node.
     */
    getRootNode(options) {
        if (!this[PropertySymbol.isConnected]) {
            return this;
        }
        if (this[PropertySymbol.rootNode] && !options?.composed) {
            return this[PropertySymbol.rootNode];
        }
        return this[PropertySymbol.ownerDocument];
    }
    /**
     * Clones a node.
     *
     * @param [deep=false] "true" to clone deep.
     * @returns Cloned node.
     */
    cloneNode(deep = false) {
        return this[PropertySymbol.cloneNode](deep);
    }
    /**
     * Append a child node to childNodes.
     *
     * @param  node Node to append.
     * @returns Appended node.
     */
    appendChild(node) {
        return this[PropertySymbol.appendChild](node);
    }
    /**
     * Remove Child element from childNodes array.
     *
     * @param node Node to remove.
     * @returns Removed node.
     */
    removeChild(node) {
        return this[PropertySymbol.removeChild](node);
    }
    /**
     * Inserts a node before another.
     *
     * @param newNode Node to insert.
     * @param referenceNode Node to insert before.
     * @returns Inserted node.
     */
    insertBefore(newNode, referenceNode) {
        if (arguments.length < 2) {
            throw new TypeError(`Failed to execute 'insertBefore' on 'Node': 2 arguments required, but only ${arguments.length} present.`);
        }
        return this[PropertySymbol.insertBefore](newNode, referenceNode);
    }
    /**
     * Replaces a node with another.
     *
     * @param newChild New child.
     * @param oldChild Old child.
     * @returns Replaced node.
     */
    replaceChild(newChild, oldChild) {
        return this[PropertySymbol.replaceChild](newChild, oldChild);
    }
    /**
     * Clones a node.
     *
     * @param [deep=false] "true" to clone deep.
     * @returns Cloned node.
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = NodeFactory.createNode(this[PropertySymbol.ownerDocument], this.constructor);
        // Document has childNodes directly when it is created
        if (clone[PropertySymbol.childNodes].length) {
            for (const node of clone[PropertySymbol.childNodes].slice()) {
                node[PropertySymbol.parentNode].removeChild(node);
            }
        }
        if (deep) {
            for (const childNode of this[PropertySymbol.childNodes]) {
                const childClone = childNode.cloneNode(true);
                childClone[PropertySymbol.parentNode] = clone;
                clone[PropertySymbol.childNodes].push(childClone);
            }
        }
        return clone;
    }
    /**
     * Append a child node to childNodes.
     *
     * @param  node Node to append.
     * @returns Appended node.
     */
    [PropertySymbol.appendChild](node) {
        return NodeUtility.appendChild(this, node);
    }
    /**
     * Remove Child element from childNodes array.
     *
     * @param node Node to remove.
     * @returns Removed node.
     */
    [PropertySymbol.removeChild](node) {
        return NodeUtility.removeChild(this, node);
    }
    /**
     * Inserts a node before another.
     *
     * @param newNode Node to insert.
     * @param referenceNode Node to insert before.
     * @returns Inserted node.
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        return NodeUtility.insertBefore(this, newNode, referenceNode);
    }
    /**
     * Replaces a node with another.
     *
     * @param newChild New child.
     * @param oldChild Old child.
     * @returns Replaced node.
     */
    [PropertySymbol.replaceChild](newChild, oldChild) {
        this.insertBefore(newChild, oldChild);
        this.removeChild(oldChild);
        return oldChild;
    }
    /**
     * Compares two nodes.
     * Two nodes are equal if they have the same type, defining the same attributes, and so on.
     *
     * @param node  Node to compare.
     * @returns boolean - `true` if two nodes are equal.
     */
    isEqualNode(node) {
        return NodeUtility.isEqualNode(this, node);
    }
    /**
     * Converts the node to a string.
     *
     * @param listener Listener.
     */
    toString() {
        return `[object ${this.constructor.name}]`;
    }
    /**
     * Observeres the node.
     * Used by MutationObserver, but it is not part of the HTML standard.
     *
     * @param listener Listener.
     */
    [PropertySymbol.observe](listener) {
        this[PropertySymbol.observers].push(listener);
        if (listener.options.subtree) {
            for (const node of this[PropertySymbol.childNodes]) {
                node[PropertySymbol.observe](listener);
            }
        }
    }
    /**
     * Stops observing the node.
     * Used by MutationObserver, but it is not part of the HTML standard.
     *
     * @param listener Listener.
     */
    [PropertySymbol.unobserve](listener) {
        const index = this[PropertySymbol.observers].indexOf(listener);
        if (index !== -1) {
            this[PropertySymbol.observers].splice(index, 1);
        }
        if (listener.options.subtree) {
            for (const node of this[PropertySymbol.childNodes]) {
                node[PropertySymbol.unobserve](listener);
            }
        }
    }
    /**
     * Connects this element to another element.
     *
     * @param parentNode Parent node.
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        const isConnected = !!parentNode && parentNode[PropertySymbol.isConnected];
        const formNode = this[PropertySymbol.formNode];
        const selectNode = this[PropertySymbol.selectNode];
        const textAreaNode = this[PropertySymbol.textAreaNode];
        const styleNode = this[PropertySymbol.styleNode];
        if (this[PropertySymbol.nodeType] !== NodeTypeEnum.documentFragmentNode) {
            this[PropertySymbol.parentNode] = parentNode;
            this[PropertySymbol.rootNode] =
                isConnected && parentNode ? parentNode[PropertySymbol.rootNode] : null;
            if (this['tagName'] !== 'FORM') {
                this[PropertySymbol.formNode] = parentNode
                    ? parentNode[PropertySymbol.formNode]
                    : null;
            }
            if (this['tagName'] !== 'SELECT') {
                this[PropertySymbol.selectNode] = parentNode
                    ? parentNode[PropertySymbol.selectNode]
                    : null;
            }
            if (this['tagName'] !== 'TEXTAREA') {
                this[PropertySymbol.textAreaNode] = parentNode
                    ? parentNode[PropertySymbol.textAreaNode]
                    : null;
            }
            if (this['tagName'] !== 'STYLE') {
                this[PropertySymbol.styleNode] = parentNode
                    ? parentNode[PropertySymbol.styleNode]
                    : null;
            }
        }
        if (this[PropertySymbol.isConnected] !== isConnected) {
            this[PropertySymbol.isConnected] = isConnected;
            if (!isConnected) {
                if (this[PropertySymbol.ownerDocument][PropertySymbol.activeElement] === this) {
                    this[PropertySymbol.ownerDocument][PropertySymbol.activeElement] = null;
                }
            }
            if (isConnected && this.connectedCallback) {
                const result = this.connectedCallback();
                /**
                 * It is common to import dependencies in the connectedCallback() method of web components.
                 * As Happy DOM doesn't have support for dynamic imports yet, this is a temporary solution to wait for imports in connectedCallback().
                 *
                 * @see https://github.com/capricorn86/happy-dom/issues/1442
                 */
                if (result instanceof Promise) {
                    const asyncTaskManager = this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow][PropertySymbol.asyncTaskManager];
                    const taskID = asyncTaskManager.startTask();
                    result
                        .then(() => asyncTaskManager.endTask(taskID))
                        .catch(() => asyncTaskManager.endTask(taskID));
                }
            }
            else if (!isConnected && this.disconnectedCallback) {
                this.disconnectedCallback();
            }
            for (const child of this[PropertySymbol.childNodes]) {
                child[PropertySymbol.connectToNode](this);
            }
            // eslint-disable-next-line
            if (this[PropertySymbol.shadowRoot]) {
                // eslint-disable-next-line
                this[PropertySymbol.shadowRoot][PropertySymbol.connectToNode](this);
            }
        }
        else if (formNode !== this[PropertySymbol.formNode] ||
            selectNode !== this[PropertySymbol.selectNode] ||
            textAreaNode !== this[PropertySymbol.textAreaNode] ||
            styleNode !== this[PropertySymbol.styleNode]) {
            for (const child of this[PropertySymbol.childNodes]) {
                child[PropertySymbol.connectToNode](this);
            }
        }
    }
    /**
     * Reports the position of its argument node relative to the node on which it is called.
     *
     * @see https://dom.spec.whatwg.org/#dom-node-comparedocumentposition
     * @param otherNode Other node.
     */
    compareDocumentPosition(otherNode) {
        /**
         * 1. If this is other, then return zero.
         */
        if (this === otherNode) {
            return 0;
        }
        /**
         * 2. Let node1 be other and node2 be this.
         */
        let node1 = otherNode;
        let node2 = this;
        /**
         * 3. Let attr1 and attr2 be null.
         */
        let attr1 = null;
        let attr2 = null;
        /**
         * 4. If node1 is an attribute, then set attr1 to node1 and node1 to attr1’s element.
         */
        if (node1[PropertySymbol.nodeType] === NodeTypeEnum.attributeNode) {
            attr1 = node1;
            node1 = attr1[PropertySymbol.ownerElement];
        }
        /**
         * 5. If node2 is an attribute, then:
         * 5.1. Set attr2 to node2 and node2 to attr2’s element.
         */
        if (node2[PropertySymbol.nodeType] === NodeTypeEnum.attributeNode) {
            attr2 = node2;
            node2 = attr2[PropertySymbol.ownerElement];
            /**
             * 5.2. If attr1 and node1 are non-null, and node2 is node1, then:
             */
            if (attr1 !== null && node1 !== null && node2 === node1) {
                /**
                 * 5.2.1. For each attr in node2’s attribute list:
                 */
                for (const attr of Object.values(node2[PropertySymbol.attributes])) {
                    /**
                     * 5.2.1.1. If attr equals attr1, then return the result of adding DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC and DOCUMENT_POSITION_PRECEDING.
                     */
                    if (NodeUtility.isEqualNode(attr, attr1)) {
                        return (Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC | Node.DOCUMENT_POSITION_PRECEDING);
                    }
                    /**
                     * 5.2.1.2. If attr equals attr2, then return the result of adding DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC and DOCUMENT_POSITION_FOLLOWING.
                     */
                    if (NodeUtility.isEqualNode(attr, attr2)) {
                        return (Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC | Node.DOCUMENT_POSITION_FOLLOWING);
                    }
                }
            }
        }
        const node2Ancestors = [];
        let node2Ancestor = node2;
        while (node2Ancestor) {
            /**
             * 7. If node1 is an ancestor of node2 […] then return the result of adding DOCUMENT_POSITION_CONTAINS to DOCUMENT_POSITION_PRECEDING.
             */
            if (node2Ancestor === node1) {
                return Node.DOCUMENT_POSITION_CONTAINS | Node.DOCUMENT_POSITION_PRECEDING;
            }
            node2Ancestors.push(node2Ancestor);
            node2Ancestor = node2Ancestor[PropertySymbol.parentNode];
        }
        const node1Ancestors = [];
        let node1Ancestor = node1;
        while (node1Ancestor) {
            /**
             * 8. If node1 is a descendant of node2 […] then return the result of adding DOCUMENT_POSITION_CONTAINED_BY to DOCUMENT_POSITION_FOLLOWING.
             */
            if (node1Ancestor === node2) {
                return Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_FOLLOWING;
            }
            node1Ancestors.push(node1Ancestor);
            node1Ancestor = node1Ancestor[PropertySymbol.parentNode];
        }
        const reverseArrayIndex = (array, reverseIndex) => {
            return array[array.length - 1 - reverseIndex];
        };
        const root = reverseArrayIndex(node2Ancestors, 0);
        /**
         * 6. If node1 or node2 is null, or node1’s root is not node2’s root, then return the result of adding
         * DOCUMENT_POSITION_DISCONNECTED, DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC, and either
         * DOCUMENT_POSITION_PRECEDING or DOCUMENT_POSITION_FOLLOWING, with the constraint that this is to be consistent, together.
         */
        if (!root || root !== reverseArrayIndex(node1Ancestors, 0)) {
            return (Node.DOCUMENT_POSITION_DISCONNECTED |
                Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC |
                Node.DOCUMENT_POSITION_FOLLOWING);
        }
        // Find the lowest common ancestor
        let commonAncestorIndex = 0;
        const ancestorsMinLength = Math.min(node2Ancestors.length, node1Ancestors.length);
        for (let i = 0; i < ancestorsMinLength; ++i) {
            const node2Ancestor = reverseArrayIndex(node2Ancestors, i);
            const node1Ancestor = reverseArrayIndex(node1Ancestors, i);
            if (node2Ancestor !== node1Ancestor) {
                break;
            }
            commonAncestorIndex = i;
        }
        const commonAncestor = reverseArrayIndex(node2Ancestors, commonAncestorIndex);
        // Indexes within the common ancestor
        let indexes = 0;
        let node2Index = -1;
        let node1Index = -1;
        const node2Node = reverseArrayIndex(node2Ancestors, commonAncestorIndex + 1);
        const node1Node = reverseArrayIndex(node1Ancestors, commonAncestorIndex + 1);
        const computeNodeIndexes = (nodes) => {
            for (const childNode of nodes) {
                computeNodeIndexes(childNode[PropertySymbol.childNodes]);
                if (childNode === node2Node) {
                    node2Index = indexes;
                }
                else if (childNode === node1Node) {
                    node1Index = indexes;
                }
                if (node2Index !== -1 && node1Index !== -1) {
                    break;
                }
                indexes++;
            }
        };
        computeNodeIndexes(commonAncestor[PropertySymbol.childNodes]);
        /**
         * 9. If node1 is preceding node2, then return DOCUMENT_POSITION_PRECEDING.
         * 10. Return DOCUMENT_POSITION_FOLLOWING.
         */
        return node1Index < node2Index
            ? Node.DOCUMENT_POSITION_PRECEDING
            : Node.DOCUMENT_POSITION_FOLLOWING;
    }
    /**
     * Normalizes the sub-tree of the node, i.e. joins adjacent text nodes, and
     * removes all empty text nodes.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Node/normalize
     */
    normalize() {
        let child = this.firstChild;
        while (child) {
            if (NodeUtility.isTextNode(child)) {
                // Append text of all following text nodes, and remove them.
                while (NodeUtility.isTextNode(child.nextSibling)) {
                    child.data += child.nextSibling.data;
                    child.nextSibling.remove();
                }
                // Remove text node if it is still empty.
                if (!child.data.length) {
                    const node = child;
                    child = child.nextSibling;
                    node.remove();
                    continue;
                }
            }
            else {
                // Normalize child nodes recursively.
                child.normalize();
            }
            child = child.nextSibling;
        }
    }
    /**
     * Determines whether the given node is equal to the current node.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Node/isSameNode
     * @param node Node to check.
     * @returns True if the given node is equal to the current node, otherwise false.
     */
    isSameNode(node) {
        return this === node;
    }
}
// Public properties
Node.ELEMENT_NODE = NodeTypeEnum.elementNode;
Node.ATTRIBUTE_NODE = NodeTypeEnum.attributeNode;
Node.TEXT_NODE = NodeTypeEnum.textNode;
Node.CDATA_SECTION_NODE = NodeTypeEnum.cdataSectionNode;
Node.COMMENT_NODE = NodeTypeEnum.commentNode;
Node.DOCUMENT_NODE = NodeTypeEnum.documentNode;
Node.DOCUMENT_TYPE_NODE = NodeTypeEnum.documentTypeNode;
Node.DOCUMENT_FRAGMENT_NODE = NodeTypeEnum.documentFragmentNode;
Node.PROCESSING_INSTRUCTION_NODE = NodeTypeEnum.processingInstructionNode;
Node.DOCUMENT_POSITION_CONTAINED_BY = NodeDocumentPositionEnum.containedBy;
Node.DOCUMENT_POSITION_CONTAINS = NodeDocumentPositionEnum.contains;
Node.DOCUMENT_POSITION_DISCONNECTED = NodeDocumentPositionEnum.disconnect;
Node.DOCUMENT_POSITION_FOLLOWING = NodeDocumentPositionEnum.following;
Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC = NodeDocumentPositionEnum.implementationSpecific;
Node.DOCUMENT_POSITION_PRECEDING = NodeDocumentPositionEnum.preceding;
export default Node;
// According to the spec, these properties should be on the prototype.
Node.prototype.ELEMENT_NODE = NodeTypeEnum.elementNode;
Node.prototype.ATTRIBUTE_NODE = NodeTypeEnum.attributeNode;
Node.prototype.TEXT_NODE = NodeTypeEnum.textNode;
Node.prototype.CDATA_SECTION_NODE = NodeTypeEnum.cdataSectionNode;
Node.prototype.COMMENT_NODE = NodeTypeEnum.commentNode;
Node.prototype.DOCUMENT_NODE = NodeTypeEnum.documentNode;
Node.prototype.DOCUMENT_TYPE_NODE = NodeTypeEnum.documentTypeNode;
Node.prototype.DOCUMENT_FRAGMENT_NODE = NodeTypeEnum.documentFragmentNode;
Node.prototype.PROCESSING_INSTRUCTION_NODE = NodeTypeEnum.processingInstructionNode;
Node.prototype.DOCUMENT_POSITION_CONTAINED_BY =
    NodeDocumentPositionEnum.containedBy;
Node.prototype.DOCUMENT_POSITION_CONTAINS =
    NodeDocumentPositionEnum.contains;
Node.prototype.DOCUMENT_POSITION_DISCONNECTED =
    NodeDocumentPositionEnum.disconnect;
Node.prototype.DOCUMENT_POSITION_FOLLOWING =
    NodeDocumentPositionEnum.following;
Node.prototype.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC =
    NodeDocumentPositionEnum.implementationSpecific;
Node.prototype.DOCUMENT_POSITION_PRECEDING =
    NodeDocumentPositionEnum.preceding;
//# sourceMappingURL=Node.js.map