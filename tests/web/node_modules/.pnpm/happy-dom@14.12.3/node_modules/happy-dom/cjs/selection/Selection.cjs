"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var _Selection_instances, _Selection_ownerDocument, _Selection_range, _Selection_direction, _Selection_associateRange;
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../event/Event.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../nodes/node/NodeTypeEnum.cjs"));
const NodeUtility_js_1 = __importDefault(require("../nodes/node/NodeUtility.cjs"));
const RangeUtility_js_1 = __importDefault(require("../range/RangeUtility.cjs"));
const SelectionDirectionEnum_js_1 = __importDefault(require("./SelectionDirectionEnum.cjs"));
/**
 * Selection.
 *
 * Based on logic from:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/selection/Selection-impl.js
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Selection.
 */
class Selection {
    /**
     * Constructor.
     *
     * @param ownerDocument Owner document.
     */
    constructor(ownerDocument) {
        _Selection_instances.add(this);
        _Selection_ownerDocument.set(this, null);
        _Selection_range.set(this, null);
        _Selection_direction.set(this, SelectionDirectionEnum_js_1.default.directionless);
        __classPrivateFieldSet(this, _Selection_ownerDocument, ownerDocument, "f");
    }
    /**
     * Returns range count.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-rangecount
     * @returns Range count.
     */
    get rangeCount() {
        return __classPrivateFieldGet(this, _Selection_range, "f") ? 1 : 0;
    }
    /**
     * Returns collapsed state.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-iscollapsed
     * @returns "true" if collapsed.
     */
    get isCollapsed() {
        return __classPrivateFieldGet(this, _Selection_range, "f") === null || __classPrivateFieldGet(this, _Selection_range, "f").collapsed;
    }
    /**
     * Returns type.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-type
     * @returns Type.
     */
    get type() {
        if (!__classPrivateFieldGet(this, _Selection_range, "f")) {
            return 'None';
        }
        else if (__classPrivateFieldGet(this, _Selection_range, "f").collapsed) {
            return 'Caret';
        }
        return 'Range';
    }
    /**
     * Returns anchor node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchornode
     * @returns Node.
     */
    get anchorNode() {
        if (!__classPrivateFieldGet(this, _Selection_range, "f")) {
            return null;
        }
        return __classPrivateFieldGet(this, _Selection_direction, "f") === SelectionDirectionEnum_js_1.default.forwards
            ? __classPrivateFieldGet(this, _Selection_range, "f").startContainer
            : __classPrivateFieldGet(this, _Selection_range, "f").endContainer;
    }
    /**
     * Returns anchor offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchoroffset
     * @returns Node.
     */
    get anchorOffset() {
        if (!__classPrivateFieldGet(this, _Selection_range, "f")) {
            return 0;
        }
        return __classPrivateFieldGet(this, _Selection_direction, "f") === SelectionDirectionEnum_js_1.default.forwards
            ? __classPrivateFieldGet(this, _Selection_range, "f").startOffset
            : __classPrivateFieldGet(this, _Selection_range, "f").endOffset;
    }
    /**
     * Returns anchor node.
     *
     * @deprecated
     * @alias anchorNode
     * @returns Node.
     */
    get baseNode() {
        return this.anchorNode;
    }
    /**
     * Returns anchor offset.
     *
     * @deprecated
     * @alias anchorOffset
     * @returns Node.
     */
    get baseOffset() {
        return this.anchorOffset;
    }
    /**
     * Returns focus node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusnode
     * @returns Node.
     */
    get focusNode() {
        return this.anchorNode;
    }
    /**
     * Returns focus offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusoffset
     * @returns Node.
     */
    get focusOffset() {
        return this.anchorOffset;
    }
    /**
     * Returns focus node.
     *
     * @deprecated
     * @alias focusNode
     * @returns Node.
     */
    get extentNode() {
        return this.focusNode;
    }
    /**
     * Returns focus offset.
     *
     * @deprecated
     * @alias focusOffset
     * @returns Node.
     */
    get extentOffset() {
        return this.focusOffset;
    }
    /**
     * Adds a range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-addrange
     * @param newRange Range.
     */
    addRange(newRange) {
        if (!newRange) {
            throw new Error('Failed to execute addRange on Selection. Parameter 1 is not of type Range.');
        }
        if (!__classPrivateFieldGet(this, _Selection_range, "f") && newRange[PropertySymbol.ownerDocument] === __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
        }
    }
    /**
     * Returns Range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-getrangeat
     * @param index Index.
     * @returns Range.
     */
    getRangeAt(index) {
        if (!__classPrivateFieldGet(this, _Selection_range, "f") || index !== 0) {
            throw new DOMException_js_1.default('Invalid range index.', DOMExceptionNameEnum_js_1.default.indexSizeError);
        }
        return __classPrivateFieldGet(this, _Selection_range, "f");
    }
    /**
     * Removes a range from a selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-removerange
     * @param range Range.
     */
    removeRange(range) {
        if (__classPrivateFieldGet(this, _Selection_range, "f") !== range) {
            throw new DOMException_js_1.default('Invalid range.', DOMExceptionNameEnum_js_1.default.notFoundError);
        }
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, null);
    }
    /**
     * Removes all ranges.
     */
    removeAllRanges() {
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, null);
    }
    /**
     * Removes all ranges.
     *
     * @alias removeAllRanges()
     */
    empty() {
        this.removeAllRanges();
    }
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapse
     * @param node Node.
     * @param offset Offset.
     */
    collapse(node, offset) {
        if (node === null) {
            this.removeAllRanges();
            return;
        }
        if (node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentTypeNode) {
            throw new DOMException_js_1.default("DocumentType Node can't be used as boundary point.", DOMExceptionNameEnum_js_1.default.invalidNodeTypeError);
        }
        if (offset > NodeUtility_js_1.default.getNodeLength(node)) {
            throw new DOMException_js_1.default('Invalid range index.', DOMExceptionNameEnum_js_1.default.indexSizeError);
        }
        if (node[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            return;
        }
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
    }
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setposition
     * @alias collapse()
     * @param node Node.
     * @param offset Offset.
     */
    setPosition(node, offset) {
        this.collapse(node, offset);
    }
    /**
     * Collapses the selection to the end.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetoend
     */
    collapseToEnd() {
        if (__classPrivateFieldGet(this, _Selection_range, "f") === null) {
            throw new DOMException_js_1.default('There is no selection to collapse.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const { node, offset } = __classPrivateFieldGet(this, _Selection_range, "f")[PropertySymbol.end];
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
    }
    /**
     * Collapses the selection to the start.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetostart
     */
    collapseToStart() {
        if (!__classPrivateFieldGet(this, _Selection_range, "f")) {
            throw new DOMException_js_1.default('There is no selection to collapse.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const { node, offset } = __classPrivateFieldGet(this, _Selection_range, "f")[PropertySymbol.start];
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
    }
    /**
     * Indicates whether a specified node is part of the selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-containsnode
     * @param node Node.
     * @param [allowPartialContainment] Set to "true" to allow partial containment.
     * @returns Always returns "true" for now.
     */
    containsNode(node, allowPartialContainment = false) {
        if (!__classPrivateFieldGet(this, _Selection_range, "f") || node[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            return false;
        }
        const startIsBeforeNode = RangeUtility_js_1.default.compareBoundaryPointsPosition(__classPrivateFieldGet(this, _Selection_range, "f")[PropertySymbol.start], {
            node,
            offset: 0
        }) === -1;
        const endIsAfterNode = RangeUtility_js_1.default.compareBoundaryPointsPosition(__classPrivateFieldGet(this, _Selection_range, "f")[PropertySymbol.end], {
            node,
            offset: NodeUtility_js_1.default.getNodeLength(node)
        }) === 1;
        return allowPartialContainment
            ? startIsBeforeNode || endIsAfterNode
            : startIsBeforeNode && endIsAfterNode;
    }
    /**
     * Deletes the selected text from the document's DOM.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-deletefromdocument
     */
    deleteFromDocument() {
        if (__classPrivateFieldGet(this, _Selection_range, "f")) {
            __classPrivateFieldGet(this, _Selection_range, "f").deleteContents();
        }
    }
    /**
     * Moves the focus of the selection to a specified point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-extend
     * @param node Node.
     * @param offset Offset.
     */
    extend(node, offset) {
        if (node[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            return;
        }
        if (!__classPrivateFieldGet(this, _Selection_range, "f")) {
            throw new DOMException_js_1.default('There is no selection to extend.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const anchorNode = this.anchorNode;
        const anchorOffset = this.anchorOffset;
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = 0;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = 0;
        if (node[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_range, "f")[PropertySymbol.ownerDocument]) {
            newRange[PropertySymbol.start].offset = offset;
            newRange[PropertySymbol.end].offset = offset;
        }
        else if (RangeUtility_js_1.default.compareBoundaryPointsPosition({ node: anchorNode, offset: anchorOffset }, { node, offset }) <= 0) {
            newRange[PropertySymbol.start].node = anchorNode;
            newRange[PropertySymbol.start].offset = anchorOffset;
            newRange[PropertySymbol.end].node = node;
            newRange[PropertySymbol.end].offset = offset;
        }
        else {
            newRange[PropertySymbol.start].node = node;
            newRange[PropertySymbol.start].offset = offset;
            newRange[PropertySymbol.end].node = anchorNode;
            newRange[PropertySymbol.end].offset = anchorOffset;
        }
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
        __classPrivateFieldSet(this, _Selection_direction, RangeUtility_js_1.default.compareBoundaryPointsPosition({ node, offset }, { node: anchorNode, offset: anchorOffset }) === -1
            ? SelectionDirectionEnum_js_1.default.backwards
            : SelectionDirectionEnum_js_1.default.forwards, "f");
    }
    /**
     * Selects all children.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-selectallchildren
     * @param node Node.
     */
    selectAllChildren(node) {
        if (node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentTypeNode) {
            throw new DOMException_js_1.default("DocumentType Node can't be used as boundary point.", DOMExceptionNameEnum_js_1.default.invalidNodeTypeError);
        }
        if (node[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            return;
        }
        const length = node.childNodes.length;
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = 0;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = length;
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
    }
    /**
     * Sets the selection to be a range including all or parts of two specified DOM nodes, and any content located between them.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setbaseandextent
     * @param anchorNode Anchor node.
     * @param anchorOffset Anchor offset.
     * @param focusNode Focus node.
     * @param focusOffset Focus offset.
     */
    setBaseAndExtent(anchorNode, anchorOffset, focusNode, focusOffset) {
        if (anchorOffset > NodeUtility_js_1.default.getNodeLength(anchorNode) ||
            focusOffset > NodeUtility_js_1.default.getNodeLength(focusNode)) {
            throw new DOMException_js_1.default('Invalid anchor or focus offset.', DOMExceptionNameEnum_js_1.default.indexSizeError);
        }
        if (anchorNode[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f") ||
            focusNode[PropertySymbol.ownerDocument] !== __classPrivateFieldGet(this, _Selection_ownerDocument, "f")) {
            return;
        }
        const anchor = { node: anchorNode, offset: anchorOffset };
        const focus = { node: focusNode, offset: focusOffset };
        const newRange = new (__classPrivateFieldGet(this, _Selection_ownerDocument, "f")[PropertySymbol.ownerWindow].Range)();
        if (RangeUtility_js_1.default.compareBoundaryPointsPosition(anchor, focus) === -1) {
            newRange[PropertySymbol.start] = anchor;
            newRange[PropertySymbol.end] = focus;
        }
        else {
            newRange[PropertySymbol.start] = focus;
            newRange[PropertySymbol.end] = anchor;
        }
        __classPrivateFieldGet(this, _Selection_instances, "m", _Selection_associateRange).call(this, newRange);
        __classPrivateFieldSet(this, _Selection_direction, RangeUtility_js_1.default.compareBoundaryPointsPosition(focus, anchor) === -1
            ? SelectionDirectionEnum_js_1.default.backwards
            : SelectionDirectionEnum_js_1.default.forwards, "f");
    }
    /**
     * Returns string currently being represented by the selection object.
     *
     * @returns Selection as string.
     */
    toString() {
        return __classPrivateFieldGet(this, _Selection_range, "f") ? __classPrivateFieldGet(this, _Selection_range, "f").toString() : '';
    }
}
_Selection_ownerDocument = new WeakMap(), _Selection_range = new WeakMap(), _Selection_direction = new WeakMap(), _Selection_instances = new WeakSet(), _Selection_associateRange = function _Selection_associateRange(range) {
    const oldRange = __classPrivateFieldGet(this, _Selection_range, "f");
    __classPrivateFieldSet(this, _Selection_range, range, "f");
    __classPrivateFieldSet(this, _Selection_direction, range === null ? SelectionDirectionEnum_js_1.default.directionless : SelectionDirectionEnum_js_1.default.forwards, "f");
    if (oldRange !== __classPrivateFieldGet(this, _Selection_range, "f")) {
        // https://w3c.github.io/selection-api/#selectionchange-event
        __classPrivateFieldGet(this, _Selection_ownerDocument, "f").dispatchEvent(new Event_js_1.default('selectionchange'));
    }
};
exports.default = Selection;
//# sourceMappingURL=Selection.cjs.map