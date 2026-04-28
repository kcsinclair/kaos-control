var _a;
import CharacterData from '../character-data/CharacterData.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Comment node.
 */
export default class Comment extends CharacterData {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum.commentNode;
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return '#comment';
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object Comment]';
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.nodeType, PropertySymbol.cloneNode)](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=Comment.js.map