var _a, _b, _c, _d, _e, _f;
import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
class Attr extends Node {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum.attributeNode;
        this[_b] = null;
        this[_c] = null;
        this[_d] = null;
        this[_e] = true;
        this[_f] = null;
    }
    /**
     * Returns specified.
     *
     * @returns Specified.
     */
    get specified() {
        return this[PropertySymbol.specified];
    }
    /**
     * Returns owner element.
     *
     * @returns Owner element.
     */
    get ownerElement() {
        return this[PropertySymbol.ownerElement];
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this[PropertySymbol.value] = value;
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
     * Returns local name.
     *
     * @returns Local name.
     */
    get localName() {
        return this[PropertySymbol.name] ? this[PropertySymbol.name].split(':').reverse()[0] : null;
    }
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix() {
        return this[PropertySymbol.name] ? this[PropertySymbol.name].split(':')[0] : null;
    }
    /**
     * @override
     */
    get textContent() {
        return this[PropertySymbol.value];
    }
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI() {
        return this[PropertySymbol.namespaceURI];
    }
}
_a = PropertySymbol.nodeType, _b = PropertySymbol.namespaceURI, _c = PropertySymbol.name, _d = PropertySymbol.value, _e = PropertySymbol.specified, _f = PropertySymbol.ownerElement;
export default Attr;
//# sourceMappingURL=Attr.js.map