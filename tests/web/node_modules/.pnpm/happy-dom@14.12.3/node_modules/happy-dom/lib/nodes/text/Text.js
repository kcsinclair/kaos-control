var _a;
import * as PropertySymbol from '../../PropertySymbol.js';
import CharacterData from '../character-data/CharacterData.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Text node.
 */
export default class Text extends CharacterData {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum.textNode;
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return '#text';
    }
    /**
     * @override
     */
    get data() {
        return this[PropertySymbol.data];
    }
    /**
     * @override
     */
    set data(data) {
        super.data = data;
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
    }
    /**
     * Breaks the Text node into two nodes at the specified offset, keeping both nodes in the tree as siblings.
     *
     * @see https://dom.spec.whatwg.org/#dom-text-splittext
     * @param offset Offset.
     * @returns New text node.
     */
    splitText(offset) {
        const length = this[PropertySymbol.data].length;
        if (offset < 0 || offset > length) {
            throw new DOMException('The index is not in the allowed range.', DOMExceptionNameEnum.indexSizeError);
        }
        const count = length - offset;
        const newData = this.substringData(offset, count);
        const newNode = this[PropertySymbol.ownerDocument].createTextNode(newData);
        if (this[PropertySymbol.parentNode] !== null) {
            this[PropertySymbol.parentNode].insertBefore(newNode, this.nextSibling);
        }
        this.replaceData(offset, count, '');
        return newNode;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object Text]';
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.nodeType, PropertySymbol.cloneNode)](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode = null) {
        const oldTextAreaNode = this[PropertySymbol.textAreaNode];
        super[PropertySymbol.connectToNode](parentNode);
        if (oldTextAreaNode !== this[PropertySymbol.textAreaNode]) {
            if (oldTextAreaNode) {
                oldTextAreaNode[PropertySymbol.resetSelection]();
            }
            if (this[PropertySymbol.textAreaNode]) {
                this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
            }
        }
    }
}
//# sourceMappingURL=Text.js.map