import Element from './Element.js';
import Node from '../node/Node.js';
import Document from '../document/Document.js';
import DocumentFragment from '../document-fragment/DocumentFragment.js';
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
    static appendChild(ancestorNode: Element | Document | DocumentFragment, node: Node, options?: {
        disableAncestorValidation?: boolean;
    }): Node;
    /**
     * Handles removing a child element from the "children" property.
     *
     * @param ancestorNode Ancestor node.
     * @param node Node.
     * @returns Removed node.
     */
    static removeChild(ancestorNode: Element | Document | DocumentFragment, node: Node): Node;
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
    static insertBefore(ancestorNode: Element | Document | DocumentFragment, newNode: Node, referenceNode: Node | null, options?: {
        disableAncestorValidation?: boolean;
    }): Node;
}
//# sourceMappingURL=ElementUtility.d.ts.map