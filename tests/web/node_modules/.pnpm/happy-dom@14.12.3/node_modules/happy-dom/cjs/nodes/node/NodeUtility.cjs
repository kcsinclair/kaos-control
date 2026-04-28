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
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("./NodeTypeEnum.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const MutationRecord_js_1 = __importDefault(require("../../mutation-observer/MutationRecord.cjs"));
const MutationTypeEnum_js_1 = __importDefault(require("../../mutation-observer/MutationTypeEnum.cjs"));
/**
 * Node utility.
 */
class NodeUtility {
    /**
     * Append a child node to childNodes.
     *
     * @param ancestorNode Ancestor node.
     * @param node Node to append.
     * @param [options] Options.
     * @param [options.disableAncestorValidation] Disables validation for checking if the node is an ancestor of the ancestorNode.
     * @returns Appended node.
     */
    static appendChild(ancestorNode, node, options) {
        if (node === ancestorNode) {
            throw new DOMException_js_1.default("Failed to execute 'appendChild' on 'Node': Not possible to append a node as a child of itself.");
        }
        if (!options?.disableAncestorValidation && this.isInclusiveAncestor(node, ancestorNode, true)) {
            throw new DOMException_js_1.default("Failed to execute 'appendChild' on 'Node': The new node is a parent of the node to insert to.", DOMExceptionNameEnum_js_1.default.domException);
        }
        // If the type is DocumentFragment, then the child nodes of if it should be moved instead of the actual node.
        // See: https://developer.mozilla.org/en-US/docs/Web/API/DocumentFragment
        if (node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentFragmentNode) {
            for (const child of node[PropertySymbol.childNodes].slice()) {
                ancestorNode.appendChild(child);
            }
            return node;
        }
        // Remove the node from its previous parent if it has any.
        if (node[PropertySymbol.parentNode]) {
            const index = node[PropertySymbol.parentNode][PropertySymbol.childNodes].indexOf(node);
            if (index !== -1) {
                node[PropertySymbol.parentNode][PropertySymbol.childNodes].splice(index, 1);
            }
        }
        if (ancestorNode[PropertySymbol.isConnected]) {
            (ancestorNode[PropertySymbol.ownerDocument] || this)[PropertySymbol.cacheID]++;
        }
        ancestorNode[PropertySymbol.childNodes].push(node);
        node[PropertySymbol.connectToNode](ancestorNode);
        // MutationObserver
        if (ancestorNode[PropertySymbol.observers].length > 0) {
            const record = new MutationRecord_js_1.default({
                target: ancestorNode,
                type: MutationTypeEnum_js_1.default.childList,
                addedNodes: [node]
            });
            for (const observer of ancestorNode[PropertySymbol.observers]) {
                if (observer.options?.subtree) {
                    node[PropertySymbol.observe](observer);
                }
                if (observer.options?.childList) {
                    observer.report(record);
                }
            }
        }
        return node;
    }
    /**
     * Remove Child element from childNodes array.
     *
     * @param ancestorNode Ancestor node.
     * @param node Node to remove.
     * @returns Removed node.
     */
    static removeChild(ancestorNode, node) {
        const index = ancestorNode[PropertySymbol.childNodes].indexOf(node);
        if (index === -1) {
            throw new DOMException_js_1.default('Failed to remove node. Node is not child of parent.');
        }
        if (ancestorNode[PropertySymbol.isConnected]) {
            (ancestorNode[PropertySymbol.ownerDocument] || this)[PropertySymbol.cacheID]++;
        }
        ancestorNode[PropertySymbol.childNodes].splice(index, 1);
        node[PropertySymbol.connectToNode](null);
        // MutationObserver
        if (ancestorNode[PropertySymbol.observers].length > 0) {
            const record = new MutationRecord_js_1.default({
                target: ancestorNode,
                type: MutationTypeEnum_js_1.default.childList,
                removedNodes: [node]
            });
            for (const observer of ancestorNode[PropertySymbol.observers]) {
                if (observer.options?.subtree) {
                    node[PropertySymbol.unobserve](observer);
                }
                if (observer.options?.childList) {
                    observer.report(record);
                }
            }
        }
        return node;
    }
    /**
     * Inserts a node before another.
     *
     * @param ancestorNode Ancestor node.
     * @param newNode Node to insert.
     * @param referenceNode Node to insert before.
     * @param [options] Options.
     * @param [options.disableAncestorValidation] Disables validation for checking if the node is an ancestor of the ancestorNode.
     * @returns Inserted node.
     */
    static insertBefore(ancestorNode, newNode, referenceNode, options) {
        if (!options?.disableAncestorValidation &&
            this.isInclusiveAncestor(newNode, ancestorNode, true)) {
            throw new DOMException_js_1.default("Failed to execute 'insertBefore' on 'Node': The new node is a parent of the node to insert to.", DOMExceptionNameEnum_js_1.default.domException);
        }
        // If the type is DocumentFragment, then the child nodes of if it should be moved instead of the actual node.
        // See: https://developer.mozilla.org/en-US/docs/Web/API/DocumentFragment
        if (newNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentFragmentNode) {
            for (const child of newNode[PropertySymbol.childNodes].slice()) {
                ancestorNode.insertBefore(child, referenceNode);
            }
            return newNode;
        }
        // If the referenceNode is null or undefined, then the newNode should be appended to the ancestorNode.
        // According to spec only null is valid, but browsers support undefined as well.
        if (!referenceNode) {
            ancestorNode.appendChild(newNode);
            return newNode;
        }
        if (ancestorNode[PropertySymbol.childNodes].indexOf(referenceNode) === -1) {
            throw new DOMException_js_1.default("Failed to execute 'insertBefore' on 'Node': The node before which the new node is to be inserted is not a child of this node.");
        }
        if (ancestorNode[PropertySymbol.isConnected]) {
            (ancestorNode[PropertySymbol.ownerDocument] || this)[PropertySymbol.cacheID]++;
        }
        if (newNode[PropertySymbol.parentNode]) {
            const index = newNode[PropertySymbol.parentNode][PropertySymbol.childNodes].indexOf(newNode);
            if (index !== -1) {
                newNode[PropertySymbol.parentNode][PropertySymbol.childNodes].splice(index, 1);
            }
        }
        ancestorNode[PropertySymbol.childNodes].splice(ancestorNode[PropertySymbol.childNodes].indexOf(referenceNode), 0, newNode);
        newNode[PropertySymbol.connectToNode](ancestorNode);
        // MutationObserver
        if (ancestorNode[PropertySymbol.observers].length > 0) {
            const record = new MutationRecord_js_1.default({
                target: ancestorNode,
                type: MutationTypeEnum_js_1.default.childList,
                addedNodes: [newNode]
            });
            for (const observer of ancestorNode[PropertySymbol.observers]) {
                if (observer.options?.subtree) {
                    newNode[PropertySymbol.observe](observer);
                }
                if (observer.options?.childList) {
                    observer.report(record);
                }
            }
        }
        return newNode;
    }
    /**
     * Returns whether the passed node is a text node, and narrows its type.
     *
     * @param node The node to be tested.
     * @returns "true" if the node is a text node.
     */
    static isTextNode(node) {
        return node?.[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.textNode;
    }
    /**
     * Returns boolean indicating if "ancestorNode" is an inclusive ancestor of "referenceNode".
     *
     * Based on:
     * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
     *
     * @see https://dom.spec.whatwg.org/#concept-tree-inclusive-ancestor
     * @param ancestorNode Ancestor node.
     * @param referenceNode Reference node.
     * @param [includeShadowRoots = false] Include shadow roots.
     * @returns "true" if inclusive ancestor.
     */
    static isInclusiveAncestor(ancestorNode, referenceNode, includeShadowRoots = false) {
        if (ancestorNode === null || referenceNode === null) {
            return false;
        }
        if (ancestorNode === referenceNode) {
            return true;
        }
        if (!ancestorNode[PropertySymbol.childNodes].length) {
            return false;
        }
        if (includeShadowRoots &&
            referenceNode[PropertySymbol.isConnected] !== ancestorNode[PropertySymbol.isConnected]) {
            return false;
        }
        if (includeShadowRoots &&
            ancestorNode === referenceNode[PropertySymbol.ownerDocument] &&
            referenceNode[PropertySymbol.isConnected]) {
            return true;
        }
        let parent = referenceNode[PropertySymbol.parentNode];
        while (parent) {
            if (ancestorNode === parent) {
                return true;
            }
            parent = parent[PropertySymbol.parentNode]
                ? parent[PropertySymbol.parentNode]
                : includeShadowRoots && parent.host
                    ? parent.host
                    : null;
        }
        return false;
    }
    /**
     * Returns boolean indicating if nodeB is following nodeA in the document tree.
     *
     * Based on:
     * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
     *
     * @see https://dom.spec.whatwg.org/#concept-tree-following
     * @param nodeA Node A.
     * @param nodeB Node B.
     * @returns "true" if following.
     */
    static isFollowing(nodeA, nodeB) {
        if (nodeA === nodeB) {
            return false;
        }
        let current = nodeB;
        while (current) {
            current = this.following(current);
            if (current === nodeA) {
                return true;
            }
        }
        return false;
    }
    /**
     * Node length.
     *
     * Based on:
     * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
     *
     * @see https://dom.spec.whatwg.org/#concept-node-length
     * @param node Node.
     * @returns Node length.
     */
    static getNodeLength(node) {
        switch (node[PropertySymbol.nodeType]) {
            case NodeTypeEnum_js_1.default.documentTypeNode:
                return 0;
            case NodeTypeEnum_js_1.default.textNode:
            case NodeTypeEnum_js_1.default.processingInstructionNode:
            case NodeTypeEnum_js_1.default.commentNode:
                return node.data.length;
            default:
                return node[PropertySymbol.childNodes].length;
        }
    }
    /**
     * Returns boolean indicating if nodeB is following nodeA in the document tree.
     *
     * Based on:
     * https://github.com/jsdom/js-symbol-tree/blob/master/lib/SymbolTree.js#L220
     *
     * @param node Node.
     * @param [root] Root.
     * @returns Following node.
     */
    static following(node, root) {
        const firstChild = node.firstChild;
        if (firstChild) {
            return firstChild;
        }
        let current = node;
        while (current) {
            if (current === root) {
                return null;
            }
            const nextSibling = current.nextSibling;
            if (nextSibling) {
                return nextSibling;
            }
            current = current[PropertySymbol.parentNode];
        }
        return null;
    }
    /**
     * Returns the next sibling or parents sibling.
     *
     * @param node Node.
     * @returns Next descendant node.
     */
    static nextDescendantNode(node) {
        while (node && !node.nextSibling) {
            node = node[PropertySymbol.parentNode];
        }
        if (!node) {
            return null;
        }
        return node.nextSibling;
    }
    /**
     * Needed by https://dom.spec.whatwg.org/#concept-node-equals
     *
     * @param elementA
     * @param elementB
     */
    static attributeListsEqual(elementA, elementB) {
        for (let i = 0, max = elementA[PropertySymbol.attributes].length; i < max; i++) {
            const attributeA = elementA[PropertySymbol.attributes][i];
            const attributeB = elementB[PropertySymbol.attributes].getNamedItemNS(attributeA[PropertySymbol.namespaceURI], attributeA.localName);
            if (!attributeB || attributeB[PropertySymbol.value] !== attributeA[PropertySymbol.value]) {
                return false;
            }
        }
        return true;
    }
    /**
     * Check if node nodeA equals node nodeB.
     * Reference: https://dom.spec.whatwg.org/#concept-node-equals
     *
     * @param nodeA Node A.
     * @param nodeB Node B.
     */
    static isEqualNode(nodeA, nodeB) {
        if (nodeA[PropertySymbol.nodeType] !== nodeB[PropertySymbol.nodeType]) {
            return false;
        }
        switch (nodeA[PropertySymbol.nodeType]) {
            case NodeTypeEnum_js_1.default.documentTypeNode:
                const documentTypeA = nodeA;
                const documentTypeB = nodeB;
                if (documentTypeA.name !== documentTypeB.name ||
                    documentTypeA.publicId !== documentTypeB.publicId ||
                    documentTypeA.systemId !== documentTypeB.systemId) {
                    return false;
                }
                break;
            case NodeTypeEnum_js_1.default.elementNode:
                const elementA = nodeA;
                const elementB = nodeB;
                if (elementA[PropertySymbol.namespaceURI] !== elementB[PropertySymbol.namespaceURI] ||
                    elementA[PropertySymbol.prefix] !== elementB[PropertySymbol.prefix] ||
                    elementA.localName !== elementB.localName ||
                    elementA[PropertySymbol.attributes].length !== elementB[PropertySymbol.attributes].length) {
                    return false;
                }
                break;
            case NodeTypeEnum_js_1.default.attributeNode:
                const attributeA = nodeA;
                const attributeB = nodeB;
                if (attributeA[PropertySymbol.namespaceURI] !== attributeB[PropertySymbol.namespaceURI] ||
                    attributeA.localName !== attributeB.localName ||
                    attributeA[PropertySymbol.value] !== attributeB[PropertySymbol.value]) {
                    return false;
                }
                break;
            case NodeTypeEnum_js_1.default.processingInstructionNode:
                const processingInstructionA = nodeA;
                const processingInstructionB = nodeB;
                if (processingInstructionA.target !== processingInstructionB.target ||
                    processingInstructionA.data !== processingInstructionB.data) {
                    return false;
                }
                break;
            case NodeTypeEnum_js_1.default.textNode:
            case NodeTypeEnum_js_1.default.commentNode:
                const textOrCommentA = nodeA;
                const textOrCommentB = nodeB;
                if (textOrCommentA.data !== textOrCommentB.data) {
                    return false;
                }
                break;
        }
        if (nodeA[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.elementNode &&
            !NodeUtility.attributeListsEqual(nodeA, nodeB)) {
            return false;
        }
        if (nodeA[PropertySymbol.childNodes].length !==
            nodeB[PropertySymbol.childNodes].length) {
            return false;
        }
        for (let i = 0; i < nodeA[PropertySymbol.childNodes].length; i++) {
            const childNodeA = nodeA[PropertySymbol.childNodes][i];
            const childNodeB = nodeB[PropertySymbol.childNodes][i];
            if (!NodeUtility.isEqualNode(childNodeA, childNodeB)) {
                return false;
            }
        }
        return true;
    }
}
exports.default = NodeUtility;
//# sourceMappingURL=NodeUtility.cjs.map