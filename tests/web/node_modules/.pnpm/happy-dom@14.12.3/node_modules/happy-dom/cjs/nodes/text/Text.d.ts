import * as PropertySymbol from '../../PropertySymbol.cjs';
import CharacterData from '../character-data/CharacterData.cjs';
import Node from '../node/Node.cjs';
import NodeTypeEnum from '../node/NodeTypeEnum.cjs';
/**
 * Text node.
 */
export default class Text extends CharacterData {
    cloneNode: (deep?: boolean) => Text;
    [PropertySymbol.nodeType]: NodeTypeEnum;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName(): string;
    /**
     * @override
     */
    get data(): string;
    /**
     * @override
     */
    set data(data: string);
    /**
     * Breaks the Text node into two nodes at the specified offset, keeping both nodes in the tree as siblings.
     *
     * @see https://dom.spec.whatwg.org/#dom-text-splittext
     * @param offset Offset.
     * @returns New text node.
     */
    splitText(offset: number): Text;
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString(): string;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): Text;
    /**
     * @override
     */
    [PropertySymbol.connectToNode](parentNode?: Node): void;
}
//# sourceMappingURL=Text.d.ts.map