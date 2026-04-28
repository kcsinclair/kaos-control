"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const NodeList_js_1 = __importDefault(require("../node/NodeList.cjs"));
/**
 * RadioNodeList
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RadioNodeList
 */
class RadioNodeList extends NodeList_js_1.default {
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
exports.default = RadioNodeList;
//# sourceMappingURL=RadioNodeList.cjs.map