var _a;
import CharacterData from '../character-data/CharacterData.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * Processing instruction node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/ProcessingInstruction.
 */
class ProcessingInstruction extends CharacterData {
    constructor() {
        super(...arguments);
        this[_a] = NodeTypeEnum.processingInstructionNode;
    }
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target() {
        return this[PropertySymbol.target];
    }
}
_a = PropertySymbol.nodeType, PropertySymbol.target;
export default ProcessingInstruction;
//# sourceMappingURL=ProcessingInstruction.js.map