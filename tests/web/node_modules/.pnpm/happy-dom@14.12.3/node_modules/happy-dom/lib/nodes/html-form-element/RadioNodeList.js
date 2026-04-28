import NodeList from '../node/NodeList.js';
/**
 * RadioNodeList
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RadioNodeList
 */
export default class RadioNodeList extends NodeList {
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        for (const node of this) {
            if (node.checked) {
                return node.value;
            }
        }
        return null;
    }
}
//# sourceMappingURL=RadioNodeList.js.map