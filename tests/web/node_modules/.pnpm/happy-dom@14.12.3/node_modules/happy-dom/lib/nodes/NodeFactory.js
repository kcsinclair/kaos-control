/**
 * Node factory used for setting the owner document to nodes.
 */
class NodeFactory {
    /**
     * Creates a node instance with the given owner document.
     *
     * @param ownerDocument Owner document.
     * @param nodeClass Node class.
     * @param [args] Node arguments.
     * @returns Node instance.
     */
    static createNode(ownerDocument, nodeClass, ...args) {
        this.ownerDocuments.push(ownerDocument);
        return new nodeClass(...args);
    }
    /**
     * Pulls an owner document from the queue.
     *
     * @returns Document.
     */
    static pullOwnerDocument() {
        return this.ownerDocuments.pop();
    }
}
NodeFactory.ownerDocuments = [];
export default NodeFactory;
//# sourceMappingURL=NodeFactory.js.map