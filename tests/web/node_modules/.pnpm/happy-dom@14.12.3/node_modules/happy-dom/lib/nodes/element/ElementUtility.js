import NodeTypeEnum from '../node/NodeTypeEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeUtility from '../node/NodeUtility.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
const NAMED_ITEM_ATTRIBUTES = ['id', 'name'];
/**
 * Element utility.
 */
export default class ElementUtility {
    /**
     * Handles appending a child element to the "children" property.
     *
     * @param ancestorNode Ancestor node.
     * @param node Node to append.
     * @param [options] Options.
     * @param [options.disableAncestorValidation] Disables validation for checking if the node is an ancestor of the ancestorNode.
     * @returns Appended node.
     */
    static appendChild(ancestorNode, node, options) {
        if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode && node !== ancestorNode) {
            if (!options?.disableAncestorValidation &&
                NodeUtility.isInclusiveAncestor(node, ancestorNode)) {
                throw new DOMException("Failed to execute 'appendChild' on 'Node': The new node is a parent of the node to insert to.", DOMExceptionNameEnum.domException);
            }
            if (node[PropertySymbol.parentNode]) {
                const parentNodeChildren = (node[PropertySymbol.parentNode][PropertySymbol.children]);
                if (parentNodeChildren) {
                    const index = parentNodeChildren.indexOf(node);
                    if (index !== -1) {
                        for (const attributeName of NAMED_ITEM_ATTRIBUTES) {
                            const attribute = node[PropertySymbol.attributes].getNamedItem(attributeName);
                            if (attribute) {
                                parentNodeChildren[PropertySymbol.removeNamedItem](node, attribute[PropertySymbol.value]);
                            }
                        }
                        parentNodeChildren.splice(index, 1);
                    }
                }
            }
            const ancestorNodeChildren = (ancestorNode[PropertySymbol.children]);
            for (const attributeName of NAMED_ITEM_ATTRIBUTES) {
                const attribute = node[PropertySymbol.attributes].getNamedItem(attributeName);
                if (attribute) {
                    ancestorNodeChildren[PropertySymbol.appendNamedItem](node, attribute[PropertySymbol.value]);
                }
            }
            ancestorNodeChildren.push(node);
            NodeUtility.appendChild(ancestorNode, node, { disableAncestorValidation: true });
        }
        else {
            NodeUtility.appendChild(ancestorNode, node, options);
        }
        return node;
    }
    /**
     * Handles removing a child element from the "children" property.
     *
     * @param ancestorNode Ancestor node.
     * @param node Node.
     * @returns Removed node.
     */
    static removeChild(ancestorNode, node) {
        if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
            const ancestorNodeChildren = (ancestorNode[PropertySymbol.children]);
            const index = ancestorNodeChildren.indexOf(node);
            if (index !== -1) {
                for (const attributeName of NAMED_ITEM_ATTRIBUTES) {
                    const attribute = node[PropertySymbol.attributes].getNamedItem(attributeName);
                    if (attribute) {
                        ancestorNodeChildren[PropertySymbol.removeNamedItem](node, attribute[PropertySymbol.value]);
                    }
                }
                ancestorNodeChildren.splice(index, 1);
            }
        }
        NodeUtility.removeChild(ancestorNode, node);
        return node;
    }
    /**
     *
     * Handles inserting a child element to the "children" property.
     *
     * @param ancestorNode Ancestor node.
     * @param newNode Node to insert.
     * @param referenceNode Node to insert before.
     * @param [options] Options.
     * @param [options.disableAncestorValidation] Disables validation for checking if the node is an ancestor of the ancestorNode.
     * @returns Inserted node.
     */
    static insertBefore(ancestorNode, newNode, referenceNode, options) {
        if (newNode === referenceNode) {
            return newNode;
        }
        // NodeUtility.insertBefore() will call appendChild() for the scenario where "referenceNode" is "null" or "undefined"
        if (newNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode && referenceNode) {
            if (!options?.disableAncestorValidation &&
                NodeUtility.isInclusiveAncestor(newNode, ancestorNode)) {
                throw new DOMException("Failed to execute 'insertBefore' on 'Node': The new node is a parent of the node to insert to.", DOMExceptionNameEnum.domException);
            }
            if (newNode[PropertySymbol.parentNode]) {
                const parentNodeChildren = (newNode[PropertySymbol.parentNode][PropertySymbol.children]);
                if (parentNodeChildren) {
                    const index = parentNodeChildren.indexOf(newNode);
                    if (index !== -1) {
                        for (const attributeName of NAMED_ITEM_ATTRIBUTES) {
                            const attribute = newNode[PropertySymbol.attributes].getNamedItem(attributeName);
                            if (attribute) {
                                parentNodeChildren[PropertySymbol.removeNamedItem](newNode, attribute[PropertySymbol.value]);
                            }
                        }
                        parentNodeChildren.splice(index, 1);
                    }
                }
                const parentChildNodes = ancestorNode[PropertySymbol.childNodes];
                if (parentChildNodes) {
                    const index = parentChildNodes.indexOf(newNode);
                    if (index !== -1) {
                        parentChildNodes.splice(index, 1);
                    }
                }
            }
            const ancestorNodeChildren = (ancestorNode[PropertySymbol.children]);
            if (referenceNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
                const index = ancestorNodeChildren.indexOf(referenceNode);
                if (index !== -1) {
                    ancestorNodeChildren.splice(index, 0, newNode);
                }
            }
            else {
                ancestorNodeChildren.length = 0;
                for (const node of ancestorNode[PropertySymbol.childNodes]) {
                    if (node === referenceNode) {
                        ancestorNodeChildren.push(newNode);
                    }
                    if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
                        ancestorNodeChildren.push(node);
                    }
                }
            }
            for (const attributeName of NAMED_ITEM_ATTRIBUTES) {
                const attribute = newNode[PropertySymbol.attributes].getNamedItem(attributeName);
                if (attribute) {
                    ancestorNodeChildren[PropertySymbol.appendNamedItem](newNode, attribute[PropertySymbol.value]);
                }
            }
            NodeUtility.insertBefore(ancestorNode, newNode, referenceNode, {
                disableAncestorValidation: true
            });
        }
        else {
            NodeUtility.insertBefore(ancestorNode, newNode, referenceNode, options);
        }
        return newNode;
    }
}
//# sourceMappingURL=ElementUtility.js.map