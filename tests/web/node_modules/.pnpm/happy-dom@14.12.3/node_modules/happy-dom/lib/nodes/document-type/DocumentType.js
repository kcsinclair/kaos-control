var _a, _b, _c, _d;
import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * DocumentType.
 */
export default class DocumentType extends Node {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum.documentTypeNode;
        this[_b] = '';
        this[_c] = '';
        this[_d] = '';
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns public ID.
     *
     * @returns Public ID.
     */
    get publicId() {
        return this[PropertySymbol.publicId];
    }
    /**
     * Returns system ID.
     *
     * @returns System ID.
     */
    get systemId() {
        return this[PropertySymbol.systemId];
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return this.name;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object DocumentType]';
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.nodeType, _b = PropertySymbol.name, _c = PropertySymbol.publicId, _d = PropertySymbol.systemId, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.name] = this[PropertySymbol.name];
        clone[PropertySymbol.publicId] = this[PropertySymbol.publicId];
        clone[PropertySymbol.systemId] = this[PropertySymbol.systemId];
        return clone;
    }
}
//# sourceMappingURL=DocumentType.js.map