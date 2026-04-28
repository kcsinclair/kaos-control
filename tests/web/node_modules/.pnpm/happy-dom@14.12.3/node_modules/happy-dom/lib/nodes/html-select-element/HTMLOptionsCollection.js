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
var _HTMLOptionsCollection_selectElement;
import DOMException from '../../exception/DOMException.js';
import HTMLCollection from '../element/HTMLCollection.js';
/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
class HTMLOptionsCollection extends HTMLCollection {
    /**
     *
     * @param selectElement
     */
    constructor(selectElement) {
        super();
        _HTMLOptionsCollection_selectElement.set(this, void 0);
        __classPrivateFieldSet(this, _HTMLOptionsCollection_selectElement, selectElement, "f");
    }
    /**
     * Returns selectedIndex.
     *
     * @returns SelectedIndex.
     */
    get selectedIndex() {
        return __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").selectedIndex;
    }
    /**
     * Sets selectedIndex.
     *
     * @param selectedIndex SelectedIndex.
     */
    set selectedIndex(selectedIndex) {
        __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").selectedIndex = selectedIndex;
    }
    /**
     * Returns item by index.
     *
     * @param index Index.
     */
    item(index) {
        return this[index];
    }
    /**
     *
     * @param element
     * @param before
     */
    add(element, before) {
        if (!before && before !== 0) {
            __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").appendChild(element);
            return;
        }
        if (!Number.isNaN(Number(before))) {
            if (before < 0) {
                return;
            }
            __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").insertBefore(element, this[before]);
            return;
        }
        const index = this.indexOf(before);
        if (index === -1) {
            throw new DOMException("Failed to execute 'add' on 'DOMException': The node before which the new node is to be inserted is not a child of this node.");
        }
        __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").insertBefore(element, this[index]);
    }
    /**
     * Removes indexed element from collection.
     *
     * @param index Index.
     */
    remove(index) {
        if (this[index]) {
            __classPrivateFieldGet(this, _HTMLOptionsCollection_selectElement, "f").removeChild(this[index]);
        }
    }
}
_HTMLOptionsCollection_selectElement = new WeakMap();
export default HTMLOptionsCollection;
//# sourceMappingURL=HTMLOptionsCollection.js.map