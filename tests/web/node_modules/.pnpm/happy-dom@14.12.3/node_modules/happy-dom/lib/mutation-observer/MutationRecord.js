/**
 * MutationRecord is a model for a mutation.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationRecord
 */
export default class MutationRecord {
    /**
     * Constructor.
     *
     * @param init Options to initialize the mutation record.
     */
    constructor(init) {
        this.type = null;
        this.target = null;
        this.addedNodes = [];
        this.removedNodes = [];
        this.previousSibling = null;
        this.nextSibling = null;
        this.attributeName = null;
        this.attributeNamespace = null;
        this.oldValue = null;
        Object.assign(this, init);
    }
}
//# sourceMappingURL=MutationRecord.js.map