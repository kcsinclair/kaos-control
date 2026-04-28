"use strict";
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _NodeIterator_walker;
Object.defineProperty(exports, "__esModule", { value: true });
const TreeWalker_js_1 = __importDefault(require("./TreeWalker.cjs"));
/**
 * The NodeIterator object represents the nodes of a document subtree and a position within them.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/NodeIterator
 */
class NodeIterator {
    /**
     * Constructor.
     *
     * @param root Root.
     * @param [whatToShow] What to show.
     * @param [filter] Filter.
     */
    constructor(root, whatToShow = -1, filter = null) {
        this.root = null;
        this.whatToShow = -1;
        this.filter = null;
        _NodeIterator_walker.set(this, void 0);
        this.root = root;
        this.whatToShow = whatToShow;
        this.filter = filter;
        __classPrivateFieldSet(this, _NodeIterator_walker, new TreeWalker_js_1.default(root, whatToShow, filter), "f");
    }
    /**
     * Moves the current Node to the next visible node in the document order.
     *
     * @returns Current node.
     */
    nextNode() {
        return __classPrivateFieldGet(this, _NodeIterator_walker, "f").nextNode();
    }
    /**
     * Moves the current Node to the previous visible node in the document order, and returns the found node. It also moves the current node to this one. If no such node exists, or if it is before that the root node defined at the object construction, returns null and the current node is not changed.
     *
     * @returns Current node.
     */
    previousNode() {
        return __classPrivateFieldGet(this, _NodeIterator_walker, "f").previousNode();
    }
}
_NodeIterator_walker = new WeakMap();
exports.default = NodeIterator;
//# sourceMappingURL=NodeIterator.cjs.map